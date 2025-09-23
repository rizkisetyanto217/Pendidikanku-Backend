// file: internals/features/school/sessions/sessions/controller/user_attendance_controller.go
package controller

import (
	"strings"

	attDTO "masjidku_backend/internals/features/school/sessions/sessions/dto"
	attModel "masjidku_backend/internals/features/school/sessions/sessions/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* =================================================================================
   LIST / GET BY ID
   ================================================================================= */

func (ctl *UserAttendanceController) List(c *fiber.Ctx) error {
	// ✅ Resolve masjid context
	mc, er := helperAuth.ResolveMasjidContext(c)
	if er != nil {
		return helper.JsonError(c, er.(*fiber.Error).Code, er.Error())
	}

	// ✅ Tentukan masjidID + authorize
	var masjidID uuid.UUID
	if helperAuth.IsOwner(c) || helperAuth.IsDKM(c) {
		id, er := helperAuth.EnsureMasjidAccessDKM(c, mc)
		if er != nil {
			return helper.JsonError(c, er.(*fiber.Error).Code, er.Error())
		}
		masjidID = id
	} else {
		switch {
		case mc.ID != uuid.Nil:
			masjidID = mc.ID
		case strings.TrimSpace(mc.Slug) != "":
			id, er := helperAuth.GetMasjidIDBySlug(c, mc.Slug)
			if er != nil {
				return helper.JsonError(c, fiber.StatusNotFound, "Masjid (slug) tidak ditemukan")
			}
			masjidID = id
		default:
			if id, er := helperAuth.GetActiveMasjidID(c); er == nil && id != uuid.Nil {
				masjidID = id
			}
		}
		if masjidID == uuid.Nil || !helperAuth.UserHasMasjid(c, masjidID) {
			return helper.JsonError(c, fiber.StatusForbidden, "Anda tidak terdaftar pada masjid ini (membership).")
		}
	}

	// include flags
	includeParam := strings.ToLower(strings.TrimSpace(c.Query("include")))
	includeURLs := includeParam == "urls"
	if !includeURLs {
		legacy := strings.ToLower(strings.TrimSpace(c.Query("include_urls")))
		includeURLs = legacy == "1" || legacy == "true" || legacy == "yes"
	}

	// --- GET BY ID mode ---
	if rawID := firstNonEmpty(
		strings.TrimSpace(c.Params("id")),
		strings.TrimSpace(c.Query("id")),
	); rawID != "" {
		id, perr := uuid.Parse(rawID)
		if perr != nil || id == uuid.Nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid")
		}

		var m attModel.UserAttendanceModel
		if err := ctl.DB.WithContext(c.Context()).
			Where(`
				user_attendance_id = ?
				AND user_attendance_masjid_id = ?
				AND user_attendance_deleted_at IS NULL
			`, id, masjidID).
			First(&m).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
		}

		// Tanpa mapper: kembalikan model langsung
		if !includeURLs {
			return helper.JsonOK(c, "OK", m)
		}
		// (opsional) load URLs di sini kalau kamu mau. Untuk sekarang hanya bungkus.
		return helper.JsonOK(c, "OK", fiber.Map{
			"attendance": m,
			// "urls": urls, // kalau nanti di-load
		})
	}

	// --- LIST mode ---
	p := helper.ParseFiber(c, "created_at", "desc", helper.AdminOpts)
	allowedOrder := map[string]string{
		"id":         "user_attendance_id",
		"created_at": "user_attendance_created_at",
	}
	orderClause, err := p.SafeOrderClause(allowedOrder, "created_at")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "sort_by tidak valid")
	}

	var q attDTO.ListUserAttendanceQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}

	tx, err := ctl.buildListQuery(c, q, masjidID)
	if err != nil {
		return err
	}

	// filter id list (opsional)
	if raw := strings.TrimSpace(c.Query("ids")); raw != "" {
		parts := strings.Split(raw, ",")
		ids := make([]uuid.UUID, 0, len(parts))
		for _, s := range parts {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			u, e := uuid.Parse(s)
			if e != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid: "+s)
			}
			ids = append(ids, u)
		}
		if len(ids) > 0 {
			tx = tx.Where("user_attendance_id IN ?", ids)
		}
	}

	// Sorting & total
	tx = tx.Order(strings.TrimPrefix(orderClause, "ORDER BY "))
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Page window
	qdb := tx
	if !p.All {
		qdb = qdb.Limit(p.Limit()).Offset(p.Offset())
	}

	var rows []attModel.UserAttendanceModel
	if err := qdb.Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	if !includeURLs {
		meta := helper.BuildMeta(total, p)
		return helper.JsonList(c, rows, fiber.Map{
			"meta":   meta,
			"total":  total,
			"limit":  p.PerPage,
			"offset": p.Offset(),
		})
	}

	items := make([]fiber.Map, 0, len(rows))
	for i := range rows {
		items = append(items, fiber.Map{
			"attendance": rows[i],
			// "urls": urlsByAttendance[rows[i].UserAttendanceID] // kalau nanti di-load batch
		})
	}

	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, items, fiber.Map{
		"meta":   meta,
		"total":  total,
		"limit":  p.PerPage,
		"offset": p.Offset(),
	})
}

/*
=================================================================================

	CREATE — POST /user-attendance
	Body: dto.UserAttendanceCreateRequest
	=================================================================================
*/
func (ctl *UserAttendanceController) Create(c *fiber.Ctx) error {
	mc, er := helperAuth.ResolveMasjidContext(c)
	if er != nil {
		return helper.JsonError(c, er.(*fiber.Error).Code, er.Error())
	}
	if !(helperAuth.IsOwner(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak")
	}

	var req attDTO.UserAttendanceCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid: "+err.Error())
	}
	// konsistenkan masjid dari context jika ada
	if mc.ID != uuid.Nil && req.MasjidID != mc.ID {
		req.MasjidID = mc.ID
	}
	if req.MasjidID == uuid.Nil || req.SessionID == uuid.Nil || req.MasjidStudentID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "masjid_id, session_id, masjid_student_id wajib diisi")
	}

	var out attModel.UserAttendanceModel
	if err := ctl.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// 1) create attendance
		m := req.ToModel()
		if err := tx.Create(&m).Error; err != nil {
			return err
		}
		// 2) URL create
		muts, err := attDTO.BuildURLMutations(m.UserAttendanceID, req.MasjidID, req.URLs)
		if err != nil {
			return err
		}
		if len(muts.ToCreate) > 0 {
			if err := tx.Create(&muts.ToCreate).Error; err != nil {
				return err
			}
		}
		// 3) normalize primary
		if err := ensurePrimaryUnique(tx, m.UserAttendanceID); err != nil {
			return err
		}
		out = m
		return nil
	}); err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// langsung return model
	return helper.JsonCreated(c, "Created", out)
}

/* ========================= Helpers ========================= */

// util kecil
func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
