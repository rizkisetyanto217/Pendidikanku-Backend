package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	attDTO  "masjidku_backend/internals/features/school/classes/class_attendance_sessions/dto"
	attModel "masjidku_backend/internals/features/school/classes/class_attendance_sessions/model"

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
	helperOSS "masjidku_backend/internals/helpers/oss"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserAttendanceController struct {
	DB        *gorm.DB
	Validator *validator.Validate
}

func NewUserAttendanceController(db *gorm.DB) *UserAttendanceController {
	return &UserAttendanceController{
		DB:        db,
		Validator: validator.New(),
	}
}

func isDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "duplicate key") ||
		strings.Contains(s, "violates unique constraint") ||
		strings.Contains(s, "unique constraint") ||
		strings.Contains(s, "sqlstate 23505")
}

const dateLayout = "2006-01-02"

// ===============================
// Helpers
// ===============================

// Pastikan session milik masjid ini (tenant-safe)
// NOTE: Tabel session tetap pakai nama lama (class_attendance_sessions)
func (ctl *UserAttendanceController) ensureSessionBelongsToMasjid(c *fiber.Ctx, sessionID, masjidID uuid.UUID) error {
	var count int64
	if err := ctl.DB.WithContext(c.Context()).
		Table("class_attendance_sessions").
		Where("class_attendance_sessions_id = ? AND class_attendance_sessions_masjid_id = ? AND class_attendance_sessions_deleted_at IS NULL",
			sessionID, masjidID).
		Count(&count).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if count == 0 {
		return fiber.NewError(fiber.StatusForbidden, "Session tidak ditemukan/diizinkan untuk masjid ini")
	}
	return nil
}

// Build list query (tenant-aware) — disesuaikan ke DTO baru
func (ctl *UserAttendanceController) buildListQuery(c *fiber.Ctx, q attDTO.ListUserClassSessionAttendanceQuery, masjidID uuid.UUID) (*gorm.DB, error) {
	tx := ctl.DB.WithContext(c.Context()).
		Model(&attModel.UserClassSessionAttendanceModel{}).
		Where("user_class_session_attendance_masjid_id = ? AND user_class_session_attendance_deleted_at IS NULL", masjidID)

	// Search di desc / notes
	if s := strings.TrimSpace(q.Search); s != "" {
		like := "%" + s + "%"
		tx = tx.Where(`
			COALESCE(user_class_session_attendance_desc,'') ILIKE ? OR
			COALESCE(user_class_session_attendance_user_note,'') ILIKE ? OR
			COALESCE(user_class_session_attendance_teacher_note,'') ILIKE ?
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
			tx = tx.Where("user_class_session_attendance_status IN ?", valid)
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
			tx = tx.Where("user_class_session_attendance_method IN ?", valid)
		}
	}

	// Filter ID (string → uuid)
	if s := strings.TrimSpace(q.SessionID); s != "" {
		if id, e := uuid.Parse(s); e == nil {
			tx = tx.Where("user_class_session_attendance_session_id = ?", id)
		} else {
			return nil, helper.JsonError(c, fiber.StatusBadRequest, "session_id tidak valid")
		}
	}
	if s := strings.TrimSpace(q.MasjidStudentID); s != "" {
		if id, e := uuid.Parse(s); e == nil {
			tx = tx.Where("user_class_session_attendance_masjid_student_id = ?", id)
		} else {
			return nil, helper.JsonError(c, fiber.StatusBadRequest, "masjid_student_id tidak valid")
		}
	}
	if s := strings.TrimSpace(q.TypeID); s != "" {
		if id, e := uuid.Parse(s); e == nil {
			tx = tx.Where("user_class_session_attendance_type_id = ?", id)
		} else {
			return nil, helper.JsonError(c, fiber.StatusBadRequest, "type_id tidak valid")
		}
	}
	if s := strings.TrimSpace(q.MarkedByTeacherID); s != "" {
		if id, e := uuid.Parse(s); e == nil {
			tx = tx.Where("user_class_session_attendance_marked_by_teacher_id = ?", id)
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
		tx = tx.Where("user_class_session_attendance_created_at >= ?", t)
	}
	if s := strings.TrimSpace(q.CreatedLE); s != "" {
		t, err := time.Parse(dateLayout, s)
		if err != nil {
			return nil, helper.JsonError(c, fiber.StatusBadRequest, "created_le invalid format, expected YYYY-MM-DD")
		}
		tx = tx.Where("user_class_session_attendance_created_at < ?", t.Add(24*time.Hour))
	}

	// Rentang waktu marked_at
	if s := strings.TrimSpace(q.MarkedGE); s != "" {
		t, err := time.Parse(dateLayout, s)
		if err != nil {
			return nil, helper.JsonError(c, fiber.StatusBadRequest, "marked_ge invalid format, expected YYYY-MM-DD")
		}
		tx = tx.Where("user_class_session_attendance_marked_at IS NOT NULL AND user_class_session_attendance_marked_at >= ?", t)
	}
	if s := strings.TrimSpace(q.MarkedLE); s != "" {
		t, err := time.Parse(dateLayout, s)
		if err != nil {
			return nil, helper.JsonError(c, fiber.StatusBadRequest, "marked_le invalid format, expected YYYY-MM-DD")
		}
		tx = tx.Where("user_class_session_attendance_marked_at IS NOT NULL AND user_class_session_attendance_marked_at < ?", t.Add(24*time.Hour))
	}

	// default order
	return tx.Order("user_class_session_attendance_created_at DESC"), nil
}

// ===============================
// Handlers
// ===============================

/*
=========================================================
POST /user-attendance (WITH URLs)
- JSON:
  {
    "attendance": { ...UserClassSessionAttendanceCreateRequest... },
    "urls": [ {op:"upsert", kind,label,href,object_key,order,is_primary,...}, ... ]
  }

- multipart/form-data:
  - attendance_json: JSON UserClassSessionAttendanceCreateRequest (wajib)
  - urls_json: JSON array UserClassSessionAttendanceURLOpDTO (opsional; op akan dipaksa "upsert")
  - file uploads: otomatis upload ke OSS → tiap file jadi URL op upsert baru (kind=attachment)
=========================================================
*/
func (ctl *UserAttendanceController) CreateWithURLs(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)
	var masjidID uuid.UUID

	// resolve masjid
	if mc, err := helperAuth.ResolveMasjidContext(c); err == nil && (mc.ID != uuid.Nil || strings.TrimSpace(mc.Slug) != "") {
		if id, er := helperAuth.EnsureMasjidAccessDKM(c, mc); er == nil {
			masjidID = id
		} else {
			if fe, ok := er.(*fiber.Error); ok {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusForbidden, er.Error())
		}
	} else {
		if id, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c); err == nil && id != uuid.Nil {
			masjidID = id
		} else {
			return helper.JsonError(c, fiber.StatusForbidden, "Scope masjid tidak ditemukan")
		}
	}

	ct := strings.ToLower(strings.TrimSpace(c.Get("Content-Type")))

	// ----- Parse payload ke DTO baru -----
	var attReq attDTO.UserClassSessionAttendanceCreateRequest
	var urlOps []attDTO.UserClassSessionAttendanceURLOpDTO

	if strings.HasPrefix(ct, "multipart/form-data") {
		aj := strings.TrimSpace(c.FormValue("attendance_json"))
		if aj == "" {
			return helper.JsonError(c, fiber.StatusBadRequest, "attendance_json wajib diisi (UserClassSessionAttendanceCreateRequest)")
		}
		if err := json.Unmarshal([]byte(aj), &attReq); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "attendance_json tidak valid: "+err.Error())
		}

		if uj := strings.TrimSpace(c.FormValue("urls_json")); uj != "" {
			_ = json.Unmarshal([]byte(uj), &urlOps)
		}
		// paksa semua op → upsert
		for i := range urlOps {
			urlOps[i].Op = attDTO.URLOpUpsert
		}

		// files → setiap file jadi URL op upsert baru (kind=attachment)
		if form, ferr := c.MultipartForm(); ferr == nil && form != nil {
			fhs, _ := helperOSS.CollectUploadFiles(form, nil)
			if len(fhs) > 0 {
				oss, oerr := helperOSS.NewOSSServiceFromEnv("")
				if oerr != nil {
					return helper.JsonError(c, fiber.StatusBadGateway, "OSS tidak siap")
				}
				ctx := context.Background()
				for _, fh := range fhs {
					publicURL, uerr := helperOSS.UploadAnyToOSS(ctx, oss, masjidID, "user_attendance", fh)
					if uerr != nil {
						return uerr
					}
					var key *string
					if k, kerr := helperOSS.ExtractKeyFromPublicURL(publicURL); kerr == nil {
						key = &k
					}
					op := attDTO.UserClassSessionAttendanceURLOpDTO{
						Op:        attDTO.URLOpUpsert,
						Kind:      ptrStr("attachment"),
						Href:      &publicURL,
						ObjectKey: key,
					}
					urlOps = append(urlOps, op)
				}
			}
		}
	} else {
		// JSON murni
		var body struct {
			Attendance attDTO.UserClassSessionAttendanceCreateRequest `json:"attendance"`
			URLs       []attDTO.UserClassSessionAttendanceURLOpDTO    `json:"urls"`
		}
		raw := bytes.TrimSpace(c.Body())
		if len(raw) == 0 {
			return helper.JsonError(c, fiber.StatusBadRequest, "Payload kosong")
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
		}
		attReq = body.Attendance
		urlOps = body.URLs
		// paksa op=upsert untuk create
		for i := range urlOps {
			urlOps[i].Op = attDTO.URLOpUpsert
			urlOps[i].ID = nil
		}
	}

	// Validasi request
	if err := ctl.Validator.Struct(&attReq); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Tenant guard: session harus milik masjid
	if err := ctl.ensureSessionBelongsToMasjid(c, attReq.SessionID, masjidID); err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Set masjid ke request
	attReq.MasjidID = masjidID

	// =========================
	// Transaksi
	// =========================
	var created attModel.UserClassSessionAttendanceModel

	if err := ctl.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// 1) create attendance
		m := attReq.ToModel()
		if err := tx.Create(&m).Error; err != nil {
			if isDuplicateKey(err) {
				return fiber.NewError(fiber.StatusConflict, "Kehadiran sudah tercatat (duplikat)")
			}
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}

		// 2) URLs via URLMutations (create only)
		muts, err := attDTO.BuildURLMutations(m.UserClassSessionAttendanceID, masjidID, urlOps)
		if err != nil {
			return err
		}
		if len(muts.ToCreate) > 0 {
			if err := tx.Create(&muts.ToCreate).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan lampiran")
			}
		}

		// 3) enforce primary uniqueness per (attendance, kind)
		if err := ensurePrimaryUnique(tx, m.UserClassSessionAttendanceID); err != nil {
			return err
		}

		created = m
		return nil
	}); err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Ambil URLs (live) untuk response
	var urls []attModel.UserClassSessionAttendanceURLModel
	_ = ctl.DB.
		Where("user_class_session_attendance_url_attendance_id = ? AND user_class_session_attendance_url_deleted_at IS NULL", created.UserClassSessionAttendanceID).
		Order("user_class_session_attendance_url_is_primary DESC, user_class_session_attendance_url_order ASC, user_class_session_attendance_url_created_at ASC").
		Find(&urls)

	c.Set("Location", "/user-attendance/"+created.UserClassSessionAttendanceID.String())
	return helper.JsonCreated(c, "Kehadiran & lampiran berhasil dibuat", fiber.Map{
		"attendance": created,
		"urls":       urls,
	})
}

/*
=========================================================
PATCH /user-attendance/:id?  (atau body.attendance_id)
Body JSON: attDTO.UserClassSessionAttendancePatchRequest
- Tri-state attendance fields
- URLs ops: [{op:"upsert"|"delete", id?, kind?, ...}]
Multipart (opsional):
- patch_json: JSON UserClassSessionAttendancePatchRequest
- files[]: tiap file akan ditambahkan sebagai URL op "upsert" baru (kind=attachment)
=========================================================
*/
func (ctl *UserAttendanceController) Patch(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	// Resolve masjid
	var masjidID uuid.UUID
	if mc, err := helperAuth.ResolveMasjidContext(c); err == nil && (mc.ID != uuid.Nil || strings.TrimSpace(mc.Slug) != "") {
		if id, er := helperAuth.EnsureMasjidAccessDKM(c, mc); er == nil {
			masjidID = id
		} else {
			if fe, ok := er.(*fiber.Error); ok {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusForbidden, er.Error())
		}
	} else {
		if id, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c); err == nil && id != uuid.Nil {
			masjidID = id
		} else {
			return helper.JsonError(c, fiber.StatusForbidden, "Scope masjid tidak ditemukan")
		}
	}

	var req attDTO.UserClassSessionAttendancePatchRequest
	ct := strings.ToLower(strings.TrimSpace(c.Get("Content-Type")))

	if strings.HasPrefix(ct, "multipart/form-data") {
		payload := strings.TrimSpace(c.FormValue("patch_json"))
		if payload == "" {
			return helper.JsonError(c, fiber.StatusBadRequest, "patch_json wajib diisi pada multipart")
		}
		if err := json.Unmarshal([]byte(payload), &req); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "patch_json tidak valid: "+err.Error())
		}

		// files[] → append sebagai URL op upsert (insert baru)
		if form, ferr := c.MultipartForm(); ferr == nil && form != nil {
			fhs, _ := helperOSS.CollectUploadFiles(form, nil)
			if len(fhs) > 0 {
				oss, oerr := helperOSS.NewOSSServiceFromEnv("")
				if oerr != nil {
					return helper.JsonError(c, fiber.StatusBadGateway, "OSS tidak siap")
				}
				ctx := context.Background()
				for _, fh := range fhs {
					publicURL, uerr := helperOSS.UploadAnyToOSS(ctx, oss, masjidID, "user_attendance", fh)
					if uerr != nil {
						return uerr
					}
					var key *string
					if k, kerr := helperOSS.ExtractKeyFromPublicURL(publicURL); kerr == nil {
						key = &k
					}
					req.URLs = append(req.URLs, attDTO.UserClassSessionAttendanceURLOpDTO{
						Op:        attDTO.URLOpUpsert,
						Kind:      ptrStr("attachment"),
						Href:      &publicURL,
						ObjectKey: key,
					})
				}
			}
		}
	} else {
		// JSON
		raw := bytes.TrimSpace(c.Body())
		if len(raw) == 0 {
			return helper.JsonError(c, fiber.StatusBadRequest, "Payload kosong")
		}
		if err := json.Unmarshal(raw, &req); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
		}
	}

	// attendance_id dari path kalau belum ada
	if req.AttendanceID == uuid.Nil {
		if s := strings.TrimSpace(c.Params("id")); s != "" {
			if id, e := uuid.Parse(s); e == nil {
				req.AttendanceID = id
			}
		}
	}
	if req.AttendanceID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "attendance_id wajib diisi")
	}

	// ── Transaksi ──
	if err := ctl.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// load + FOR UPDATE (tenant guard)
		var m attModel.UserClassSessionAttendanceModel
		q := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_class_session_attendance_id = ? AND user_class_session_attendance_deleted_at IS NULL", req.AttendanceID)
		if masjidID != uuid.Nil {
			q = q.Where("user_class_session_attendance_masjid_id = ?", masjidID)
		}
		if err := q.First(&m).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
			}
			return err
		}

		// apply tri-state patch
		if err := req.ApplyPatch(&m); err != nil {
			return err
		}
		if err := tx.Save(&m).Error; err != nil {
			return err
		}

		// URL ops → mutations
		muts, err := attDTO.BuildURLMutations(m.UserClassSessionAttendanceID, m.UserClassSessionAttendanceMasjidID, req.URLs)
		if err != nil {
			return err
		}

		// create
		if len(muts.ToCreate) > 0 {
			if err := tx.Create(&muts.ToCreate).Error; err != nil {
				return err
			}
		}
		// update (merge partial)
		for _, u := range muts.ToUpdate {
			var cur attModel.UserClassSessionAttendanceURLModel
			if err := tx.
				Where("user_class_session_attendance_url_id = ? AND user_class_session_attendance_url_deleted_at IS NULL", u.UserClassSessionAttendanceURLID).
				First(&cur).Error; err != nil {
				return err
			}
			mergeURL(&cur, &u)
			if err := tx.Save(&cur).Error; err != nil {
				return err
			}
		}
		// delete (soft)
		if len(muts.ToDelete) > 0 {
			if err := tx.Model(&attModel.UserClassSessionAttendanceURLModel{}).
				Where("user_class_session_attendance_url_id IN ?", muts.ToDelete).
				Update("user_class_session_attendance_url_deleted_at", gorm.Expr("NOW()")).Error; err != nil {
				return err
			}
		}

		// normalize primary unik per (attendance, kind)
		if err := ensurePrimaryUnique(tx, m.UserClassSessionAttendanceID); err != nil {
			return err
		}
		return nil
	}); err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Balikan state terbaru
	var urls []attModel.UserClassSessionAttendanceURLModel
	_ = ctl.DB.
		Where("user_class_session_attendance_url_attendance_id = ? AND user_class_session_attendance_url_deleted_at IS NULL", req.AttendanceID).
		Order("user_class_session_attendance_url_is_primary DESC, user_class_session_attendance_url_order ASC, user_class_session_attendance_url_created_at ASC").
		Find(&urls)

	return helper.JsonUpdated(c, "Attendance berhasil di-update", fiber.Map{
		"attendance_id": req.AttendanceID,
		"urls":          urls,
	})
}

/* ========================= URL Delete (tetap) ========================= */

func (ctl *UserAttendanceController) Delete(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	// Resolve masjid
	var masjidID uuid.UUID
	if mc, err := helperAuth.ResolveMasjidContext(c); err == nil && (mc.ID != uuid.Nil || strings.TrimSpace(mc.Slug) != "") {
		if id, er := helperAuth.EnsureMasjidAccessDKM(c, mc); er == nil {
			masjidID = id
		} else {
			if fe, ok := er.(*fiber.Error); ok {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusForbidden, er.Error())
		}
	} else {
		if id, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c); err == nil && id != uuid.Nil {
			masjidID = id
		} else {
			return helper.JsonError(c, fiber.StatusForbidden, "Scope masjid tidak ditemukan")
		}
	}

	attID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "attendance id tidak valid")
	}
	urlID, err := uuid.Parse(strings.TrimSpace(c.Params("url_id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "url id tidak valid")
	}

	// Ambil row, pastikan tenant & owner benar
	var row attModel.UserClassSessionAttendanceURLModel
	if err := ctl.DB.WithContext(c.Context()).
		Where(`
			user_class_session_attendance_url_id = ?
			AND user_class_session_attendance_url_attendance_id = ?
			AND user_class_session_attendance_url_masjid_id = ?
			AND user_class_session_attendance_url_deleted_at IS NULL
		`, urlID, attID, masjidID).
		Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "URL tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Opsional: pindahkan object aktif ke spam/
	var trashURL *string
	if row.UserClassSessionAttendanceURLHref != nil && strings.TrimSpace(*row.UserClassSessionAttendanceURLHref) != "" {
		if moved, mErr := helperOSS.MoveToSpamByPublicURLENV(*row.UserClassSessionAttendanceURLHref, 0); mErr == nil && strings.TrimSpace(moved) != "" {
			trashURL = &moved
		}
	}

	// Retention
	retentionDays := 30
	if v := strings.TrimSpace(strings.ToLower(os.Getenv("RETENTION_DAYS"))); v != "" {
		if n, e := strconv.Atoi(v); e == nil && n > 0 {
			retentionDays = n
		}
	}
	cutoff := time.Now().Add(time.Duration(retentionDays) * 24 * time.Hour)

	// Soft-delete
	if err := ctl.DB.WithContext(c.Context()).
		Model(&attModel.UserClassSessionAttendanceURLModel{}).
		Where("user_class_session_attendance_url_id = ?", row.UserClassSessionAttendanceURLID).
		Updates(map[string]any{
			"user_class_session_attendance_url_deleted_at":           time.Now(),
			"user_class_session_attendance_url_trash_url":            trashURL,
			"user_class_session_attendance_url_delete_pending_until": cutoff,
			"user_class_session_attendance_url_updated_at":           time.Now(),
		}).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus URL")
	}

	return helper.JsonDeleted(c, "Lampiran dihapus (soft-delete)", fiber.Map{
		"attendance_id": attID,
		"url_id":        urlID,
		"trash_url":     trashURL,
		"purge_after":   cutoff,
	})
}

/* ========================= Internals ========================= */

func ensurePrimaryUnique(tx *gorm.DB, attendanceID uuid.UUID) error {
	var primaries []struct {
		Kind string
		ID   uuid.UUID
	}
	if err := tx.
		Model(&attModel.UserClassSessionAttendanceURLModel{}).
		Select("user_class_session_attendance_url_kind AS kind, MIN(user_class_session_attendance_url_id) AS id").
		Where("user_class_session_attendance_url_attendance_id = ? AND user_class_session_attendance_url_deleted_at IS NULL AND user_class_session_attendance_url_is_primary = TRUE", attendanceID).
		Group("user_class_session_attendance_url_kind").
		Scan(&primaries).Error; err != nil {
		return err
	}
	for _, pk := range primaries {
		if err := tx.Model(&attModel.UserClassSessionAttendanceURLModel{}).
			Where(`
				user_class_session_attendance_url_attendance_id = ?
				AND user_class_session_attendance_url_kind = ?
				AND user_class_session_attendance_url_deleted_at IS NULL
				AND user_class_session_attendance_url_id <> ?
			`, attendanceID, pk.Kind, pk.ID).
			Update("user_class_session_attendance_url_is_primary", false).Error; err != nil {
			return err
		}
	}
	return nil
}

func mergeURL(cur *attModel.UserClassSessionAttendanceURLModel, patch *attModel.UserClassSessionAttendanceURLModel) {
	if patch.UserClassSessionAttendanceURLKind != "" {
		cur.UserClassSessionAttendanceURLKind = patch.UserClassSessionAttendanceURLKind
	}
	if patch.UserClassSessionAttendanceURLLabel != nil {
		cur.UserClassSessionAttendanceURLLabel = patch.UserClassSessionAttendanceURLLabel
	}
	cur.UserClassSessionAttendanceURLOrder = patch.UserClassSessionAttendanceURLOrder
	cur.UserClassSessionAttendanceURLIsPrimary = patch.UserClassSessionAttendanceURLIsPrimary

	if patch.UserClassSessionAttendanceURLHref != nil {
		cur.UserClassSessionAttendanceURLHref = patch.UserClassSessionAttendanceURLHref
	}
	if patch.UserClassSessionAttendanceURLObjectKey != nil {
		cur.UserClassSessionAttendanceURLObjectKey = patch.UserClassSessionAttendanceURLObjectKey
	}
	if patch.UserClassSessionAttendanceURLObjectKeyOld != nil {
		cur.UserClassSessionAttendanceURLObjectKeyOld = patch.UserClassSessionAttendanceURLObjectKeyOld
	}
	if patch.UserClassSessionAttendanceURLTrashURL != nil {
		cur.UserClassSessionAttendanceURLTrashURL = patch.UserClassSessionAttendanceURLTrashURL
	}
	if patch.UserClassSessionAttendanceURLDeletePendingUntil != nil {
		cur.UserClassSessionAttendanceURLDeletePendingUntil = patch.UserClassSessionAttendanceURLDeletePendingUntil
	}
	if patch.UserClassSessionAttendanceURLUploaderTeacherID != nil {
		cur.UserClassSessionAttendanceURLUploaderTeacherID = patch.UserClassSessionAttendanceURLUploaderTeacherID
	}
	if patch.UserClassSessionAttendanceURLUploaderStudentID != nil {
		cur.UserClassSessionAttendanceURLUploaderStudentID = patch.UserClassSessionAttendanceURLUploaderStudentID
	}
	cur.UserClassSessionAttendanceURLUpdatedAt = time.Now()
}

// kecil
func ptrStr(s string) *string { return &s }
