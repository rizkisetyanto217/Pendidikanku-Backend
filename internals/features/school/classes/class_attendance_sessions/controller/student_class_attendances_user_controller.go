// file: internals/features/school/sessions/sessions/controller/student_attendance_list_controller.go
package controller

import (
	"strings"
	"time"

	attDTO "schoolku_backend/internals/features/school/classes/class_attendance_sessions/dto"
	attModel "schoolku_backend/internals/features/school/classes/class_attendance_sessions/model"
	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* =================================================================================
   LIST / GET BY ID
   ================================================================================= */

// Build list query (tenant-aware) — versi student_*
func (ctl *StudentAttendanceController) buildListQuery(c *fiber.Ctx, q attDTO.ListStudentClassSessionAttendanceQuery, schoolID uuid.UUID) (*gorm.DB, error) {
	tx := ctl.DB.WithContext(c.Context()).
		Model(&attModel.StudentClassSessionAttendanceModel{}).
		Where("student_class_session_attendance_school_id = ? AND student_class_session_attendance_deleted_at IS NULL", schoolID)

	// Search di desc / notes
	if s := strings.TrimSpace(q.Search); s != "" {
		like := "%" + s + "%"
		tx = tx.Where(`
			COALESCE(student_class_session_attendance_desc,'') ILIKE ? OR
			COALESCE(student_class_session_attendance_user_note,'') ILIKE ? OR
			COALESCE(student_class_session_attendance_teacher_note,'') ILIKE ?
		`, like, like, like)
	}

	// status_in
	if len(q.StatusIn) > 0 {
		valid := make([]string, 0, len(q.StatusIn))
		for _, v := range q.StatusIn {
			vv := strings.ToLower(strings.TrimSpace(v))
			switch vv {
			case "unmarked", "present", "absent", "excused", "late":
				valid = append(valid, vv)
			}
		}
		if len(valid) > 0 {
			tx = tx.Where("student_class_session_attendance_status IN ?", valid)
		}
	}

	// method_in
	if len(q.MethodIn) > 0 {
		valid := make([]string, 0, len(q.MethodIn))
		for _, v := range q.MethodIn {
			vv := strings.ToLower(strings.TrimSpace(v))
			switch vv {
			case "manual", "qr", "geo", "import", "api", "self":
				valid = append(valid, vv)
			}
		}
		if len(valid) > 0 {
			tx = tx.Where("student_class_session_attendance_method IN ?", valid)
		}
	}

	// Filter ID (string → uuid)
	if s := strings.TrimSpace(q.SessionID); s != "" {
		if id, e := uuid.Parse(s); e == nil {
			tx = tx.Where("student_class_session_attendance_session_id = ?", id)
		} else {
			return nil, helper.JsonError(c, fiber.StatusBadRequest, "session_id tidak valid")
		}
	}
	if s := strings.TrimSpace(q.SchoolStudentID); s != "" {
		if id, e := uuid.Parse(s); e == nil {
			tx = tx.Where("student_class_session_attendance_school_student_id = ?", id)
		} else {
			return nil, helper.JsonError(c, fiber.StatusBadRequest, "school_student_id tidak valid")
		}
	}
	if s := strings.TrimSpace(q.TypeID); s != "" {
		if id, e := uuid.Parse(s); e == nil {
			tx = tx.Where("student_class_session_attendance_type_id = ?", id)
		} else {
			return nil, helper.JsonError(c, fiber.StatusBadRequest, "type_id tidak valid")
		}
	}
	if s := strings.TrimSpace(q.MarkedByTeacherID); s != "" {
		if id, e := uuid.Parse(s); e == nil {
			tx = tx.Where("student_class_session_attendance_marked_by_teacher_id = ?", id)
		} else {
			return nil, helper.JsonError(c, fiber.StatusBadRequest, "marked_by_teacher_id tidak valid")
		}
	}

	// Rentang waktu created_at
	if s := strings.TrimSpace(q.CreatedGE); s != "" {
		t, err := time.Parse(dateLayout, s)
		if err != nil {
			return nil, helper.JsonError(c, fiber.StatusBadRequest, "created_ge invalid format, expected YYYY-MM-DD")
		}
		tx = tx.Where("student_class_session_attendance_created_at >= ?", t)
	}
	if s := strings.TrimSpace(q.CreatedLE); s != "" {
		t, err := time.Parse(dateLayout, s)
		if err != nil {
			return nil, helper.JsonError(c, fiber.StatusBadRequest, "created_le invalid format, expected YYYY-MM-DD")
		}
		tx = tx.Where("student_class_session_attendance_created_at < ?", t.Add(24*time.Hour))
	}

	// Rentang waktu marked_at
	if s := strings.TrimSpace(q.MarkedGE); s != "" {
		t, err := time.Parse(dateLayout, s)
		if err != nil {
			return nil, helper.JsonError(c, fiber.StatusBadRequest, "marked_ge invalid format, expected YYYY-MM-DD")
		}
		tx = tx.Where("student_class_session_attendance_marked_at IS NOT NULL AND student_class_session_attendance_marked_at >= ?", t)
	}
	if s := strings.TrimSpace(q.MarkedLE); s != "" {
		t, err := time.Parse(dateLayout, s)
		if err != nil {
			return nil, helper.JsonError(c, fiber.StatusBadRequest, "marked_le invalid format, expected YYYY-MM-DD")
		}
		tx = tx.Where("student_class_session_attendance_marked_at IS NOT NULL AND student_class_session_attendance_marked_at < ?", t.Add(24*time.Hour))
	}

	// default order
	return tx.Order("student_class_session_attendance_created_at DESC"), nil
}

func (ctl *StudentAttendanceController) List(c *fiber.Ctx) error {
	// ✅ Resolve school context
	mc, er := helperAuth.ResolveSchoolContext(c)
	if er != nil {
		return helper.JsonError(c, er.(*fiber.Error).Code, er.Error())
	}

	// ✅ Tentukan schoolID + authorize
	var schoolID uuid.UUID
	if helperAuth.IsOwner(c) || helperAuth.IsDKM(c) {
		id, er := helperAuth.EnsureSchoolAccessDKM(c, mc)
		if er != nil {
			return helper.JsonError(c, er.(*fiber.Error).Code, er.Error())
		}
		schoolID = id
	} else {
		switch {
		case mc.ID != uuid.Nil:
			schoolID = mc.ID
		case strings.TrimSpace(mc.Slug) != "":
			id, er := helperAuth.GetSchoolIDBySlug(c, mc.Slug)
			if er != nil {
				return helper.JsonError(c, fiber.StatusNotFound, "School (slug) tidak ditemukan")
			}
			schoolID = id
		default:
			if id, er := helperAuth.GetActiveSchoolID(c); er == nil && id != uuid.Nil {
				schoolID = id
			}
		}
		if schoolID == uuid.Nil || !helperAuth.UserHasSchool(c, schoolID) {
			return helper.JsonError(c, fiber.StatusForbidden, "Anda tidak terdaftar pada school ini (membership).")
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
	rawID := strings.TrimSpace(c.Params("id"))
	if rawID == "" {
		rawID = strings.TrimSpace(c.Query("id"))
	}
	if rawID != "" {
		id, perr := uuid.Parse(rawID)
		if perr != nil || id == uuid.Nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid")
		}

		var m attModel.StudentClassSessionAttendanceModel
		if err := ctl.DB.WithContext(c.Context()).
			Where(`
			student_class_session_attendance_id = ?
			AND student_class_session_attendance_school_id = ?
			AND student_class_session_attendance_deleted_at IS NULL
		`, id, schoolID).
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
		// (opsional) load URLs di sini kalau perlu.
		return helper.JsonOK(c, "OK", fiber.Map{
			"attendance": m,
			// "urls": urls, // jika nanti di-load
		})
	}

	// --- LIST mode ---
	p := helper.ParseFiber(c, "created_at", "desc", helper.AdminOpts)
	allowedOrder := map[string]string{
		"id":         "student_class_session_attendance_id",
		"created_at": "student_class_session_attendance_created_at",
	}
	orderClause, err := p.SafeOrderClause(allowedOrder, "created_at")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "sort_by tidak valid")
	}

	var q attDTO.ListStudentClassSessionAttendanceQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}

	tx, err := ctl.buildListQuery(c, q, schoolID)
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
			tx = tx.Where("student_class_session_attendance_id IN ?", ids)
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

	var rows []attModel.StudentClassSessionAttendanceModel
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
			// "urls": urlsByAttendance[rows[i].StudentClassSessionAttendanceID] // kalau nanti di-load batch
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
CREATE — POST /student-attendance
Body: dto.StudentClassSessionAttendanceCreateRequest
=================================================================================
*/
func (ctl *StudentAttendanceController) Create(c *fiber.Ctx) error {
	mc, er := helperAuth.ResolveSchoolContext(c)
	if er != nil {
		return helper.JsonError(c, er.(*fiber.Error).Code, er.Error())
	}
	if !(helperAuth.IsOwner(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak")
	}

	var req attDTO.StudentClassSessionAttendanceCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid: "+err.Error())
	}
	// konsistenkan school dari context jika ada
	if mc.ID != uuid.Nil && req.SchoolID != mc.ID {
		req.SchoolID = mc.ID
	}
	if req.SchoolID == uuid.Nil || req.SessionID == uuid.Nil || req.SchoolStudentID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "school_id, session_id, school_student_id wajib diisi")
	}

	var out attModel.StudentClassSessionAttendanceModel
	if err := ctl.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// 1) create attendance
		m := req.ToModel()
		if err := tx.Create(&m).Error; err != nil {
			return err
		}
		// 2) URL create
		muts, err := attDTO.BuildURLMutations(m.StudentClassSessionAttendanceID, req.SchoolID, req.URLs)
		if err != nil {
			return err
		}
		if len(muts.ToCreate) > 0 {
			if err := tx.Create(&muts.ToCreate).Error; err != nil {
				return err
			}
		}
		// 3) normalize primary
		if err := ensurePrimaryUnique(tx, m.StudentClassSessionAttendanceID); err != nil {
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
