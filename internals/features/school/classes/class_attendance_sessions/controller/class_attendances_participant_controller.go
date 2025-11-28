// file: internals/features/attendance/controller/class_attendance_session_participant_controller.go
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

	attendanceDTO "madinahsalam_backend/internals/features/school/classes/class_attendance_sessions/dto"
	attendanceModel "madinahsalam_backend/internals/features/school/classes/class_attendance_sessions/model"
	attendanceService "madinahsalam_backend/internals/features/school/classes/class_attendance_sessions/service"

	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"
	helperOSS "madinahsalam_backend/internals/helpers/oss"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ClassAttendanceSessionParticipantController struct {
	DB        *gorm.DB
	Validator *validator.Validate

	PermSvc *attendanceService.AttendancePermissionService
}

func NewClassAttendanceSessionParticipantController(db *gorm.DB) *ClassAttendanceSessionParticipantController {
	return &ClassAttendanceSessionParticipantController{
		DB:        db,
		Validator: validator.New(),
		PermSvc:   attendanceService.NewAttendancePermissionService(db),
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
// Snapshot helper (murid & ortu)
// ===============================
type studentSnapshotRow struct {
	StudentName        string  `gorm:"column:student_name"`
	StudentAvatarURL   *string `gorm:"column:student_avatar_url"`
	StudentWhatsappURL *string `gorm:"column:student_whatsapp_url"`
	ParentName         *string `gorm:"column:parent_name"`
	ParentWhatsappURL  *string `gorm:"column:parent_whatsapp_url"`
	StudentGender      *string `gorm:"column:student_gender"`
	StudentCode        *string `gorm:"column:student_code"`
}

func (ctl *ClassAttendanceSessionParticipantController) hydrateStudentSnapshotForParticipant(
	ctx context.Context,
	schoolID uuid.UUID,
	req *attendanceDTO.ClassAttendanceSessionParticipantCreateRequest,
) error {
	if req.ClassAttendanceSessionParticipantSchoolStudentID == nil ||
		*req.ClassAttendanceSessionParticipantSchoolStudentID == uuid.Nil {
		return nil
	}

	// kalau sudah ada snapshot (diisi manual) → skip
	if req.ClassAttendanceSessionParticipantStudentNameSnapshot != nil {
		return nil
	}

	var row studentSnapshotRow

	err := ctl.DB.WithContext(ctx).
		Table("school_students AS ss").
		Where(`
			ss.school_student_id = ?
			AND ss.school_student_school_id = ?
			AND ss.school_student_deleted_at IS NULL
		`, *req.ClassAttendanceSessionParticipantSchoolStudentID, schoolID).
		Select(`
			COALESCE(
				ss.school_student_user_profile_name_snapshot,
				''
			) AS student_name,
			ss.school_student_user_profile_avatar_url_snapshot          AS student_avatar_url,
			ss.school_student_user_profile_whatsapp_url_snapshot        AS student_whatsapp_url,
			ss.school_student_user_profile_parent_name_snapshot         AS parent_name,
			ss.school_student_user_profile_parent_whatsapp_url_snapshot AS parent_whatsapp_url,
			ss.school_student_user_profile_gender_snapshot              AS student_gender,
			ss.school_student_code                                      AS student_code
		`).
		Take(&row).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// nggak fatal, cuma nggak ada snapshot
			return nil
		}
		return err
	}

	// isi snapshot hanya kalau kosong
	if req.ClassAttendanceSessionParticipantStudentNameSnapshot == nil && strings.TrimSpace(row.StudentName) != "" {
		req.ClassAttendanceSessionParticipantStudentNameSnapshot = &row.StudentName
	}
	if req.ClassAttendanceSessionParticipantStudentAvatarURLSnapshot == nil && row.StudentAvatarURL != nil {
		req.ClassAttendanceSessionParticipantStudentAvatarURLSnapshot = row.StudentAvatarURL
	}
	if req.ClassAttendanceSessionParticipantStudentWhatsappURLSnapshot == nil && row.StudentWhatsappURL != nil {
		req.ClassAttendanceSessionParticipantStudentWhatsappURLSnapshot = row.StudentWhatsappURL
	}
	if req.ClassAttendanceSessionParticipantParentNameSnapshot == nil && row.ParentName != nil {
		req.ClassAttendanceSessionParticipantParentNameSnapshot = row.ParentName
	}
	if req.ClassAttendanceSessionParticipantParentWhatsappURLSnapshot == nil && row.ParentWhatsappURL != nil {
		req.ClassAttendanceSessionParticipantParentWhatsappURLSnapshot = row.ParentWhatsappURL
	}
	if req.ClassAttendanceSessionParticipantStudentGenderSnapshot == nil && row.StudentGender != nil {
		req.ClassAttendanceSessionParticipantStudentGenderSnapshot = row.StudentGender
	}
	if req.ClassAttendanceSessionParticipantStudentCodeSnapshot == nil && row.StudentCode != nil {
		req.ClassAttendanceSessionParticipantStudentCodeSnapshot = row.StudentCode
	}

	return nil
}

// ===============================
// Helper: resolve school dari token
// ===============================

func (ctl *ClassAttendanceSessionParticipantController) resolveSchoolIDFromToken(c *fiber.Ctx) (uuid.UUID, error) {
	// beberapa helper auth pakai DB dari Locals
	c.Locals("DB", ctl.DB)

	// 1) ambil school_id dari token / active-school
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		// helper sudah balikin JsonError yang proper
		return uuid.Nil, err
	}

	// 2) pastikan user adalah member school ini
	if err := helperAuth.EnsureMemberSchool(c, schoolID); err != nil {
		return uuid.Nil, err
	}

	return schoolID, nil
}

// ===============================
// Handlers
// ===============================

/*
=========================================================
POST /api/u/class-attendance-session-participants (WITH URLs)
=========================================================
*/
func (ctl *ClassAttendanceSessionParticipantController) CreateAttendanceParticipantsWithURLs(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	// resolve school via token (tanpa slug)
	schoolID, err := ctl.resolveSchoolIDFromToken(c)
	if err != nil {
		return err
	}

	ct := strings.ToLower(strings.TrimSpace(c.Get("Content-Type")))

	// ----- Parse payload ke DTO baru -----
	var attReq attendanceDTO.ClassAttendanceSessionParticipantCreateRequest
	var urlOps []attendanceDTO.ClassAttendanceSessionParticipantURLOpDTO

	if strings.HasPrefix(ct, "multipart/form-data") {
		// =========================
		// MODE: multipart/form-data
		// =========================
		aj := strings.TrimSpace(c.FormValue("attendance_json"))
		if aj == "" {
			return helper.JsonError(c, fiber.StatusBadRequest, "attendance_json wajib diisi (ClassAttendanceSessionParticipantCreateRequest)")
		}
		if err := json.Unmarshal([]byte(aj), &attReq); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "attendance_json tidak valid: "+err.Error())
		}

		// 1) urls_json → URL ops biasa (link, youtube, dll)
		if uj := strings.TrimSpace(c.FormValue("urls_json")); uj != "" {
			_ = json.Unmarshal([]byte(uj), &urlOps)
		}
		for i := range urlOps {
			urlOps[i].Op = attendanceDTO.URLOpUpsert
		}

		// 2) files_meta_json → metadata untuk tiap file upload
		var fileMetas []attendanceDTO.ClassAttendanceSessionParticipantURLOpDTO
		if fm := strings.TrimSpace(c.FormValue("files_meta_json")); fm != "" {
			_ = json.Unmarshal([]byte(fm), &fileMetas)
		}

		// 3) files → upload ke OSS, merge dengan fileMetas[idx]
		if form, ferr := c.MultipartForm(); ferr == nil && form != nil {
			fhs, _ := helperOSS.CollectUploadFiles(form, nil)
			if len(fhs) > 0 {
				oss, oerr := helperOSS.NewOSSServiceFromEnv("")
				if oerr != nil {
					return helper.JsonError(c, fiber.StatusBadGateway, "OSS tidak siap")
				}
				ctx := context.Background()

				for idx, fh := range fhs {
					publicURL, uerr := helperOSS.UploadAnyToOSS(ctx, oss, schoolID, "attendance_participant", fh)
					if uerr != nil {
						return uerr
					}
					var key *string
					if k, kerr := helperOSS.ExtractKeyFromPublicURL(publicURL); kerr == nil {
						key = &k
					}

					// default DTO untuk file upload
					op := attendanceDTO.ClassAttendanceSessionParticipantURLOpDTO{
						Op:        attendanceDTO.URLOpUpsert,
						Kind:      ptrStr("attachment"),
						URL:       &publicURL,
						ObjectKey: key,
					}

					// kalau ada metadata di files_meta_json[idx], merge
					if idx < len(fileMetas) {
						meta := fileMetas[idx]

						if meta.Kind != nil {
							op.Kind = meta.Kind
						}
						if meta.Label != nil {
							op.Label = meta.Label
						}
						if meta.Order != nil {
							op.Order = meta.Order
						}
						if meta.IsPrimary != nil {
							op.IsPrimary = meta.IsPrimary
						}
						if meta.UploaderTeacherID != nil {
							op.UploaderTeacherID = meta.UploaderTeacherID
						}
						if meta.UploaderStudentID != nil {
							op.UploaderStudentID = meta.UploaderStudentID
						}
					}

					urlOps = append(urlOps, op)
				}
			}
		}
	} else {
		// =========================
		// MODE: JSON murni
		// =========================
		var body struct {
			Attendance attendanceDTO.ClassAttendanceSessionParticipantCreateRequest `json:"attendance"`
			URLs       []attendanceDTO.ClassAttendanceSessionParticipantURLOpDTO    `json:"urls"`
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
		for i := range urlOps {
			urlOps[i].Op = attendanceDTO.URLOpUpsert
			urlOps[i].ID = nil
		}
	}

	// Set school ke request (tenant)
	attReq.ClassAttendanceSessionParticipantSchoolID = schoolID

	// =========================
	// Tentukan kind + isi student/teacher ID dari token
	// =========================

	kindRaw := strVal(attReq.ClassAttendanceSessionParticipantKind)
	kind := strings.ToLower(strings.TrimSpace(kindRaw))
	if kind == "" {
		return helper.JsonError(c, fiber.StatusBadRequest,
			"class_attendance_session_participant_kind wajib diisi (student/teacher)")
	}
	attReq.ClassAttendanceSessionParticipantKind = &kind

	// cek apakah sudah diisi manual (misal admin kirim untuk orang lain)
	hasStudent := attReq.ClassAttendanceSessionParticipantSchoolStudentID != nil &&
		*attReq.ClassAttendanceSessionParticipantSchoolStudentID != uuid.Nil
	hasTeacher := attReq.ClassAttendanceSessionParticipantSchoolTeacherID != nil &&
		*attReq.ClassAttendanceSessionParticipantSchoolTeacherID != uuid.Nil

	switch kind {
	case "student":
		if !hasStudent {
			// ambil school_student_id yang terikat ke school ini dari token
			studentID, err := helperAuth.GetSchoolStudentIDForSchool(c, schoolID)
			if err != nil || studentID == uuid.Nil {
				return helper.JsonError(c, fiber.StatusForbidden,
					"Tidak dapat menentukan school_student_id dari token untuk school ini")
			}
			attReq.ClassAttendanceSessionParticipantSchoolStudentID = &studentID
			hasStudent = true
		}
	case "teacher":
		if !hasTeacher {
			// ambil school_teacher_id yang terikat ke school ini dari token
			teacherID, err := helperAuth.GetSchoolTeacherIDForSchool(c, schoolID)
			if err != nil || teacherID == uuid.Nil {
				return helper.JsonError(c, fiber.StatusForbidden,
					"Tidak dapat menentukan school_teacher_id dari token untuk school ini")
			}
			attReq.ClassAttendanceSessionParticipantSchoolTeacherID = &teacherID
			hasTeacher = true
		}
	default:
		// assistant / guest dll → boleh pakai ID yang dikirim manual,
		// di bawah tetap dicek minimal ada salah satu student/teacher jika memang diwajibkan
	}

	// minimal salah satu harus ada
	if !hasStudent && !hasTeacher {
		return helper.JsonError(c, fiber.StatusBadRequest,
			"Minimal salah satu dari school_student_id atau school_teacher_id wajib diisi (atau token harus punya konteks sesuai kind)")
	}

	// =========================
	// Guard: session_id wajib ada
	// =========================
	if attReq.ClassAttendanceSessionParticipantSessionID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest,
			"class_attendance_session_participant_session_id wajib diisi")
	}

	// =========================
	// Cek PERMISSION absensi (window, flag type, status session, mapping CSST)
	// =========================
	if ctl.PermSvc != nil {
		var studentIDPtr *uuid.UUID
		var teacherIDPtr *uuid.UUID

		if hasStudent {
			studentIDPtr = attReq.ClassAttendanceSessionParticipantSchoolStudentID
		}
		if hasTeacher {
			teacherIDPtr = attReq.ClassAttendanceSessionParticipantSchoolTeacherID
		}

		res, err := ctl.PermSvc.CheckSelfAttendancePermission(
			c.Context(),
			schoolID,
			attReq.ClassAttendanceSessionParticipantSessionID,
			kind,
			studentIDPtr,
			teacherIDPtr,
		)
		if err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengecek izin absensi: "+err.Error())
		}
		if !res.Allowed {
			// kalau mau, FE bisa bedain pakai kode res.Code
			return helper.JsonError(c, fiber.StatusForbidden, res.Message)
		}
	}

	// =========================
	// Hydrate SNAPSHOT murid (kalau ada)
	// =========================
	if hasStudent {
		if err := ctl.hydrateStudentSnapshotForParticipant(c.Context(), schoolID, &attReq); err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil snapshot siswa: "+err.Error())
		}
	}

	// =========================
	// Normalisasi waktu → UTC + default
	// =========================
	nowUTC := time.Now().UTC()

	// checkin_at
	if attReq.ClassAttendanceSessionParticipantCheckinAt == nil {
		attReq.ClassAttendanceSessionParticipantCheckinAt = &nowUTC
	} else {
		t := attReq.ClassAttendanceSessionParticipantCheckinAt.UTC()
		attReq.ClassAttendanceSessionParticipantCheckinAt = &t
	}

	// marked_at
	if attReq.ClassAttendanceSessionParticipantMarkedAt == nil {
		attReq.ClassAttendanceSessionParticipantMarkedAt = &nowUTC
	} else {
		t := attReq.ClassAttendanceSessionParticipantMarkedAt.UTC()
		attReq.ClassAttendanceSessionParticipantMarkedAt = &t
	}

	// checkout_at (kalau ada)
	if attReq.ClassAttendanceSessionParticipantCheckoutAt != nil {
		t := attReq.ClassAttendanceSessionParticipantCheckoutAt.UTC()
		attReq.ClassAttendanceSessionParticipantCheckoutAt = &t
	}

	// locked_at (kalau ada)
	if attReq.ClassAttendanceSessionParticipantLockedAt != nil {
		t := attReq.ClassAttendanceSessionParticipantLockedAt.UTC()
		attReq.ClassAttendanceSessionParticipantLockedAt = &t
	}

	// Validasi request
	if err := ctl.Validator.Struct(&attReq); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// =========================
	// Transaksi
	// =========================
	var created attendanceModel.ClassAttendanceSessionParticipantModel

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
		muts, err := attendanceDTO.BuildURLMutations(m.ClassAttendanceSessionParticipantID, schoolID, urlOps)
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
	var urls []attendanceModel.ClassAttendanceSessionParticipantURLModel
	_ = ctl.DB.
		Where("class_attendance_session_participant_url_participant_id = ? AND class_attendance_session_participant_url_deleted_at IS NULL",
			created.ClassAttendanceSessionParticipantID).
		Order("class_attendance_session_participant_url_is_primary DESC, class_attendance_session_participant_url_order ASC, class_attendance_session_participant_url_created_at ASC").
		Find(&urls)

	c.Set("Location", "/class-attendance-session-participants/"+created.ClassAttendanceSessionParticipantID.String())
	return helper.JsonCreated(c, "Kehadiran & lampiran berhasil dibuat", fiber.Map{
		"attendance": created,
		"urls":       urls,
	})
}

/*
=========================================================
PATCH /api/u/class-attendance-session-participants/:id?
=========================================================
*/
func (ctl *ClassAttendanceSessionParticipantController) Patch(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	// Resolve school dari token
	schoolID, err := ctl.resolveSchoolIDFromToken(c)
	if err != nil {
		return err
	}

	var req attendanceDTO.ClassAttendanceSessionParticipantPatchRequest
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
					publicURL, uerr := helperOSS.UploadAnyToOSS(ctx, oss, schoolID, "attendance_participant", fh)
					if uerr != nil {
						return uerr
					}
					var key *string
					if k, kerr := helperOSS.ExtractKeyFromPublicURL(publicURL); kerr == nil {
						key = &k
					}
					req.URLs = append(req.URLs, attendanceDTO.ClassAttendanceSessionParticipantURLOpDTO{
						Op:        attendanceDTO.URLOpUpsert,
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

	// class_attendance_session_participant_id dari path kalau belum ada
	if req.ClassAttendanceSessionParticipantID == uuid.Nil {
		if s := strings.TrimSpace(c.Params("id")); s != "" {
			if id, e := uuid.Parse(s); e == nil {
				req.ClassAttendanceSessionParticipantID = id
			}
		}
	}
	if req.ClassAttendanceSessionParticipantID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "class_attendance_session_participant_id wajib diisi")
	}

	// ── Transaksi ──
	if err := ctl.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// load + FOR UPDATE (tenant guard)
		var m attendanceModel.ClassAttendanceSessionParticipantModel
		q := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("class_attendance_session_participant_id = ? AND class_attendance_session_participant_deleted_at IS NULL",
				req.ClassAttendanceSessionParticipantID)
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
		muts, err := attendanceDTO.BuildURLMutations(
			m.ClassAttendanceSessionParticipantID,
			m.ClassAttendanceSessionParticipantSchoolID,
			req.URLs,
		)
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
			var cur attendanceModel.ClassAttendanceSessionParticipantURLModel
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
			if err := tx.Model(&attendanceModel.ClassAttendanceSessionParticipantURLModel{}).
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
	var urls []attendanceModel.ClassAttendanceSessionParticipantURLModel
	_ = ctl.DB.
		Where("class_attendance_session_participant_url_participant_id = ? AND class_attendance_session_participant_url_deleted_at IS NULL",
			req.ClassAttendanceSessionParticipantID).
		Order("class_attendance_session_participant_url_is_primary DESC, class_attendance_session_participant_url_order ASC, class_attendance_session_participant_url_created_at ASC").
		Find(&urls)

	return helper.JsonUpdated(c, "Attendance berhasil di-update", fiber.Map{
		"attendance_id": req.ClassAttendanceSessionParticipantID,
		"urls":          urls,
	})
}

/*
=========================================================
DELETE /api/u/class-attendance-session-participants/:id/urls/:url_id
=========================================================
*/
func (ctl *ClassAttendanceSessionParticipantController) Delete(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	// Resolve school dari token
	schoolID, err := ctl.resolveSchoolIDFromToken(c)
	if err != nil {
		return err
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
	var row attendanceModel.ClassAttendanceSessionParticipantURLModel
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
		Model(&attendanceModel.ClassAttendanceSessionParticipantURLModel{}).
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
	var urls []attendanceModel.ClassAttendanceSessionParticipantURLModel
	if err := tx.
		Where(`
            class_attendance_session_participant_url_participant_id = ?
            AND class_attendance_session_participant_url_deleted_at IS NULL
            AND class_attendance_session_participant_url_is_primary = TRUE
        `, participantID).
		Order("class_attendance_session_participant_url_created_at ASC").
		Find(&urls).Error; err != nil {
		return err
	}

	if len(urls) == 0 {
		return nil
	}

	keep := make(map[string]uuid.UUID)
	var toUnset []uuid.UUID

	for _, u := range urls {
		kind := strings.ToLower(strings.TrimSpace(u.ClassAttendanceSessionParticipantURLKind))
		if kind == "" {
			kind = "default"
		}

		if _, ok := keep[kind]; !ok {
			keep[kind] = u.ClassAttendanceSessionParticipantURLID
		} else {
			toUnset = append(toUnset, u.ClassAttendanceSessionParticipantURLID)
		}
	}

	if len(toUnset) == 0 {
		return nil
	}

	if err := tx.
		Model(&attendanceModel.ClassAttendanceSessionParticipantURLModel{}).
		Where("class_attendance_session_participant_url_id IN ?", toUnset).
		Update("class_attendance_session_participant_url_is_primary", false).Error; err != nil {
		return err
	}

	return nil
}

func mergeURL(cur *attendanceModel.ClassAttendanceSessionParticipantURLModel, patch *attendanceModel.ClassAttendanceSessionParticipantURLModel) {
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

func ptrStr(s string) *string { return &s }

// helper kecil buat ambil nilai string dari pointer
func strVal(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

/*
   ROUTE SLUG (saran)

   POST   /api/u/class-attendance-session-participants
   PATCH  /api/u/class-attendance-session-participants/:id
   DELETE /api/u/class-attendance-session-participants/:id/urls/:url_id
*/
