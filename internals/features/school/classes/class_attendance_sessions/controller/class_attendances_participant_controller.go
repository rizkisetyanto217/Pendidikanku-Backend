// file: internals/features/school/sessions/sessions/controller/student_attendance_controller.go
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

	attModel "schoolku_backend/internals/features/school/classes/class_attendance_sessions/model"
	attDTO "schoolku_backend/internals/features/school/classes/class_attendance_sessions/dto"

	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"
	helperOSS "schoolku_backend/internals/helpers/oss"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type StudentAttendanceController struct {
	DB        *gorm.DB
	Validator *validator.Validate
}

func NewStudentAttendanceController(db *gorm.DB) *StudentAttendanceController {
	return &StudentAttendanceController{
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

// Pastikan session milik school ini (tenant-safe)
// NOTE: Tabel session tetap pakai nama lama (class_attendance_sessions)
func (ctl *StudentAttendanceController) ensureSessionBelongsToSchool(c *fiber.Ctx, sessionID, schoolID uuid.UUID) error {
	var count int64
	if err := ctl.DB.WithContext(c.Context()).
		Table("class_attendance_sessions").
		Where("class_attendance_sessions_id = ? AND class_attendance_sessions_school_id = ? AND class_attendance_sessions_deleted_at IS NULL",
			sessionID, schoolID).
		Count(&count).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if count == 0 {
		return fiber.NewError(fiber.StatusForbidden, "Session tidak ditemukan/diizinkan untuk school ini")
	}
	return nil
}

// ===============================
// Handlers
// ===============================

/*
=========================================================
POST /student-attendance (WITH URLs)
  - JSON:
    {
    "attendance": { ...ClassAttendanceSessionParticipantCreateRequest... },
    "urls": [ {op:"upsert", kind,label,url,object_key,order,is_primary,...}, ... ]
    }

- multipart/form-data:
  - attendance_json: JSON ClassAttendanceSessionParticipantCreateRequest (wajib)
  - urls_json: JSON array ClassAttendanceSessionParticipantURLOpDTO (opsional; op akan dipaksa "upsert")
  - file uploads: otomatis upload ke OSS → tiap file jadi URL op upsert baru (kind=attachment)

=========================================================
*/
func (ctl *StudentAttendanceController) CreateAttendanceParticipantsWithURLs(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)
	var schoolID uuid.UUID

	// resolve school
	if mc, err := helperAuth.ResolveSchoolContext(c); err == nil && (mc.ID != uuid.Nil || strings.TrimSpace(mc.Slug) != "") {
		if id, er := helperAuth.EnsureSchoolAccessDKM(c, mc); er == nil {
			schoolID = id
		} else {
			if fe, ok := er.(*fiber.Error); ok {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusForbidden, er.Error())
		}
	} else {
		if id, err := helperAuth.GetSchoolIDFromTokenPreferTeacher(c); err == nil && id != uuid.Nil {
			schoolID = id
		} else {
			return helper.JsonError(c, fiber.StatusForbidden, "Scope school tidak ditemukan")
		}
	}

	ct := strings.ToLower(strings.TrimSpace(c.Get("Content-Type")))

	// ----- Parse payload ke DTO baru -----
	var attReq attDTO.ClassAttendanceSessionParticipantCreateRequest
	var urlOps []attDTO.ClassAttendanceSessionParticipantURLOpDTO

	if strings.HasPrefix(ct, "multipart/form-data") {
		aj := strings.TrimSpace(c.FormValue("attendance_json"))
		if aj == "" {
			return helper.JsonError(c, fiber.StatusBadRequest, "attendance_json wajib diisi (ClassAttendanceSessionParticipantCreateRequest)")
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
					publicURL, uerr := helperOSS.UploadAnyToOSS(ctx, oss, schoolID, "student_attendance", fh)
					if uerr != nil {
						return uerr
					}
					var key *string
					if k, kerr := helperOSS.ExtractKeyFromPublicURL(publicURL); kerr == nil {
						key = &k
					}
					op := attDTO.ClassAttendanceSessionParticipantURLOpDTO{
						Op:        attDTO.URLOpUpsert,
						Kind:      ptrStr("attachment"),
						URL:       &publicURL,
						ObjectKey: key,
					}
					urlOps = append(urlOps, op)
				}
			}
		}
	} else {
		// JSON murni
		var body struct {
			Attendance attDTO.ClassAttendanceSessionParticipantCreateRequest `json:"attendance"`
			URLs       []attDTO.ClassAttendanceSessionParticipantURLOpDTO    `json:"urls"`
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

	// Set school ke request (tenant)
	attReq.SchoolID = schoolID

	// Validasi request
	if err := ctl.Validator.Struct(&attReq); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Tenant guard: session harus milik school
	if err := ctl.ensureSessionBelongsToSchool(c, attReq.SessionID, schoolID); err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// =========================
	// Transaksi
	// =========================
	var created attModel.ClassAttendanceSessionParticipantModel

	if err := ctl.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// 1) create attendance (participant)
		m := attReq.ToModel()
		if err := tx.Create(&m).Error; err != nil {
			if isDuplicateKey(err) {
				return fiber.NewError(fiber.StatusConflict, "Kehadiran sudah tercatat (duplikat)")
			}
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}

		// 2) URLs via URLMutations (create only)
		muts, err := attDTO.BuildURLMutations(m.ClassAttendanceSessionParticipantID, schoolID, urlOps)
		if err != nil {
			return err
		}
		if len(muts.ToCreate) > 0 {
			if err := tx.Create(&muts.ToCreate).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan lampiran")
			}
		}

		// 3) enforce primary uniqueness per (participant, kind)
		if err := ensurePrimaryUnique(tx, m.ClassAttendanceSessionParticipantID); err != nil {
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
	var urls []attModel.ClassAttendanceSessionParticipantURLModel
	_ = ctl.DB.
		Where("class_attendance_session_participant_url_participant_id = ? AND class_attendance_session_participant_url_deleted_at IS NULL",
			created.ClassAttendanceSessionParticipantID).
		Order("class_attendance_session_participant_url_is_primary DESC, class_attendance_session_participant_url_order ASC, class_attendance_session_participant_url_created_at ASC").
		Find(&urls)

	c.Set("Location", "/student-attendance/"+created.ClassAttendanceSessionParticipantID.String())
	return helper.JsonCreated(c, "Kehadiran & lampiran berhasil dibuat", fiber.Map{
		"attendance": created,
		"urls":       urls,
	})
}

/*
=========================================================
PATCH /student-attendance/:id?  (atau body.participant_id)
Body JSON: ClassAttendanceSessionParticipantPatchRequest
- Tri-state attendance fields
- URLs ops: [{op:"upsert"|"delete", id?, kind?, ...}]
Multipart (opsional):
- patch_json: JSON ClassAttendanceSessionParticipantPatchRequest
- files[]: tiap file akan ditambahkan sebagai URL op "upsert" baru (kind=attachment)
=========================================================
*/
func (ctl *StudentAttendanceController) Patch(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	// Resolve school
	var schoolID uuid.UUID
	if mc, err := helperAuth.ResolveSchoolContext(c); err == nil && (mc.ID != uuid.Nil || strings.TrimSpace(mc.Slug) != "") {
		if id, er := helperAuth.EnsureSchoolAccessDKM(c, mc); er == nil {
			schoolID = id
		} else {
			if fe, ok := er.(*fiber.Error); ok {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusForbidden, er.Error())
		}
	} else {
		if id, err := helperAuth.GetSchoolIDFromTokenPreferTeacher(c); err == nil && id != uuid.Nil {
			schoolID = id
		} else {
			return helper.JsonError(c, fiber.StatusForbidden, "Scope school tidak ditemukan")
		}
	}

	var req attDTO.ClassAttendanceSessionParticipantPatchRequest
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
					publicURL, uerr := helperOSS.UploadAnyToOSS(ctx, oss, schoolID, "student_attendance", fh)
					if uerr != nil {
						return uerr
					}
					var key *string
					if k, kerr := helperOSS.ExtractKeyFromPublicURL(publicURL); kerr == nil {
						key = &k
					}
					req.URLs = append(req.URLs, attDTO.ClassAttendanceSessionParticipantURLOpDTO{
						Op:        attDTO.URLOpUpsert,
						Kind:      ptrStr("attachment"),
						URL:       &publicURL,
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

	// participant_id dari path kalau belum ada
	if req.ParticipantID == uuid.Nil {
		if s := strings.TrimSpace(c.Params("id")); s != "" {
			if id, e := uuid.Parse(s); e == nil {
				req.ParticipantID = id
			}
		}
	}
	if req.ParticipantID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "participant_id wajib diisi")
	}

	// ── Transaksi ──
	if err := ctl.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// load + FOR UPDATE (tenant guard)
		var m attModel.ClassAttendanceSessionParticipantModel
		q := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("class_attendance_session_participant_id = ? AND class_attendance_session_participant_deleted_at IS NULL", req.ParticipantID)
		if schoolID != uuid.Nil {
			q = q.Where("class_attendance_session_participant_school_id = ?", schoolID)
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
		muts, err := attDTO.BuildURLMutations(m.ClassAttendanceSessionParticipantID, m.ClassAttendanceSessionParticipantSchoolID, req.URLs)
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
			var cur attModel.ClassAttendanceSessionParticipantURLModel
			if err := tx.
				Where("class_attendance_session_participant_url_id = ? AND class_attendance_session_participant_url_deleted_at IS NULL",
					u.ClassAttendanceSessionParticipantURLID).
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
			if err := tx.Model(&attModel.ClassAttendanceSessionParticipantURLModel{}).
				Where("class_attendance_session_participant_url_id IN ?", muts.ToDelete).
				Update("class_attendance_session_participant_url_deleted_at", gorm.Expr("NOW()")).Error; err != nil {
				return err
			}
		}

		// normalize primary unik per (participant, kind)
		if err := ensurePrimaryUnique(tx, m.ClassAttendanceSessionParticipantID); err != nil {
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
	var urls []attModel.ClassAttendanceSessionParticipantURLModel
	_ = ctl.DB.
		Where("class_attendance_session_participant_url_participant_id = ? AND class_attendance_session_participant_url_deleted_at IS NULL",
			req.ParticipantID).
		Order("class_attendance_session_participant_url_is_primary DESC, class_attendance_session_participant_url_order ASC, class_attendance_session_participant_url_created_at ASC").
		Find(&urls)

	return helper.JsonUpdated(c, "Attendance berhasil di-update", fiber.Map{
		"attendance_id": req.ParticipantID, // tetap pakai key lama di response kalau mau backward-compatible
		"urls":          urls,
	})
}

/* ========================= URL Delete (tetap) ========================= */

func (ctl *StudentAttendanceController) Delete(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	// Resolve school
	var schoolID uuid.UUID
	if mc, err := helperAuth.ResolveSchoolContext(c); err == nil && (mc.ID != uuid.Nil || strings.TrimSpace(mc.Slug) != "") {
		if id, er := helperAuth.EnsureSchoolAccessDKM(c, mc); er == nil {
			schoolID = id
		} else {
			if fe, ok := er.(*fiber.Error); ok {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusForbidden, er.Error())
		}
	} else {
		if id, err := helperAuth.GetSchoolIDFromTokenPreferTeacher(c); err == nil && id != uuid.Nil {
			schoolID = id
		} else {
			return helper.JsonError(c, fiber.StatusForbidden, "Scope school tidak ditemukan")
		}
	}

	participantID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "participant id tidak valid")
	}
	urlID, err := uuid.Parse(strings.TrimSpace(c.Params("url_id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "url id tidak valid")
	}

	// Ambil row, pastikan tenant & owner benar
	var row attModel.ClassAttendanceSessionParticipantURLModel
	if err := ctl.DB.WithContext(c.Context()).
		Where(`
			class_attendance_session_participant_url_id = ?
			AND class_attendance_session_participant_url_participant_id = ?
			AND class_attendance_session_participant_url_school_id = ?
			AND class_attendance_session_participant_url_deleted_at IS NULL
		`, urlID, participantID, schoolID).
		Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "URL tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Opsional: pindahkan object aktif ke spam/
	var trashURL *string
	if row.ClassAttendanceSessionParticipantURL != nil && strings.TrimSpace(*row.ClassAttendanceSessionParticipantURL) != "" {
		if moved, mErr := helperOSS.MoveToSpamByPublicURLENV(*row.ClassAttendanceSessionParticipantURL, 0); mErr == nil && strings.TrimSpace(moved) != "" {
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
		Model(&attModel.ClassAttendanceSessionParticipantURLModel{}).
		Where("class_attendance_session_participant_url_id = ?", row.ClassAttendanceSessionParticipantURLID).
		Updates(map[string]any{
			"class_attendance_session_participant_url_deleted_at":           time.Now(),
			"class_attendance_session_participant_url_trash_url":            trashURL,
			"class_attendance_session_participant_url_delete_pending_until": cutoff,
			"class_attendance_session_participant_url_updated_at":           time.Now(),
		}).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus URL")
	}

	return helper.JsonDeleted(c, "Lampiran dihapus (soft-delete)", fiber.Map{
		"attendance_id": participantID,
		"url_id":        urlID,
		"trash_url":     trashURL,
		"purge_after":   cutoff,
	})
}

/* ========================= Internals ========================= */

func ensurePrimaryUnique(tx *gorm.DB, participantID uuid.UUID) error {
	var primaries []struct {
		Kind string
		ID   uuid.UUID
	}
	if err := tx.
		Model(&attModel.ClassAttendanceSessionParticipantURLModel{}).
		Select("class_attendance_session_participant_url_kind AS kind, MIN(class_attendance_session_participant_url_id) AS id").
		Where(`
			class_attendance_session_participant_url_participant_id = ?
			AND class_attendance_session_participant_url_deleted_at IS NULL
			AND class_attendance_session_participant_url_is_primary = TRUE
		`, participantID).
		Group("class_attendance_session_participant_url_kind").
		Scan(&primaries).Error; err != nil {
		return err
	}
	for _, pk := range primaries {
		if err := tx.Model(&attModel.ClassAttendanceSessionParticipantURLModel{}).
			Where(`
				class_attendance_session_participant_url_participant_id = ?
				AND class_attendance_session_participant_url_kind = ?
				AND class_attendance_session_participant_url_deleted_at IS NULL
				AND class_attendance_session_participant_url_id <> ?
			`, participantID, pk.Kind, pk.ID).
			Update("class_attendance_session_participant_url_is_primary", false).Error; err != nil {
			return err
		}
	}
	return nil
}

func mergeURL(cur *attModel.ClassAttendanceSessionParticipantURLModel, patch *attModel.ClassAttendanceSessionParticipantURLModel) {
	if patch.ClassAttendanceSessionParticipantURLKind != "" {
		cur.ClassAttendanceSessionParticipantURLKind = patch.ClassAttendanceSessionParticipantURLKind
	}
	if patch.ClassAttendanceSessionParticipantURLLabel != nil {
		cur.ClassAttendanceSessionParticipantURLLabel = patch.ClassAttendanceSessionParticipantURLLabel
	}
	cur.ClassAttendanceSessionParticipantURLOrder = patch.ClassAttendanceSessionParticipantURLOrder
	cur.ClassAttendanceSessionParticipantURLIsPrimary = patch.ClassAttendanceSessionParticipantURLIsPrimary

	if patch.ClassAttendanceSessionParticipantURL != nil {
		cur.ClassAttendanceSessionParticipantURL = patch.ClassAttendanceSessionParticipantURL
	}
	if patch.ClassAttendanceSessionParticipantURLObjectKey != nil {
		cur.ClassAttendanceSessionParticipantURLObjectKey = patch.ClassAttendanceSessionParticipantURLObjectKey
	}
	if patch.ClassAttendanceSessionParticipantURLOld != nil {
		cur.ClassAttendanceSessionParticipantURLOld = patch.ClassAttendanceSessionParticipantURLOld
	}
	if patch.ClassAttendanceSessionParticipantURLObjectKeyOld != nil {
		cur.ClassAttendanceSessionParticipantURLObjectKeyOld = patch.ClassAttendanceSessionParticipantURLObjectKeyOld
	}
	if patch.ClassAttendanceSessionParticipantURLDeletePendingUntil != nil {
		cur.ClassAttendanceSessionParticipantURLDeletePendingUntil = patch.ClassAttendanceSessionParticipantURLDeletePendingUntil
	}
	if patch.ClassAttendanceSessionParticipantURLUploaderTeacherID != nil {
		cur.ClassAttendanceSessionParticipantURLUploaderTeacherID = patch.ClassAttendanceSessionParticipantURLUploaderTeacherID
	}
	if patch.ClassAttendanceSessionParticipantURLUploaderStudentID != nil {
		cur.ClassAttendanceSessionParticipantURLUploaderStudentID = patch.ClassAttendanceSessionParticipantURLUploaderStudentID
	}
	cur.ClassAttendanceSessionParticipantURLUpdatedAt = time.Now()
}

// kecil
func ptrStr(s string) *string { return &s }
