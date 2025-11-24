// file: internals/features/school/class_schedules/controller/class_schedule_controller.go
package controller

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/lib/pq"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	sessModel "madinahsalam_backend/internals/features/school/classes/class_attendance_sessions/model"
	d "madinahsalam_backend/internals/features/school/classes/class_schedules/dto"
	m "madinahsalam_backend/internals/features/school/classes/class_schedules/model"
	svc "madinahsalam_backend/internals/features/school/classes/class_schedules/services"
)

/* =========================
   Controller & Constructor
   ========================= */

type ClassScheduleController struct {
	DB       *gorm.DB
	Validate *validator.Validate
}

func New(db *gorm.DB, v *validator.Validate) *ClassScheduleController {
	return &ClassScheduleController{DB: db, Validate: v}
}

/* =========================
   Helpers
   ========================= */

func parseUUIDParam(c *fiber.Ctx, name string) (uuid.UUID, error) {
	idStr := strings.TrimSpace(c.Params(name))
	if idStr == "" {
		return uuid.Nil, fmt.Errorf("%s is required", name)
	}
	return uuid.Parse(idStr)
}

// Ambil school_id aktif dari token (teacher/admin/dkm) dan pastikan konsisten dengan body.
// Jika token tidak punya scope school, dilepas (public).
func enforceSchoolScopeAuth(c *fiber.Ctx, bodySchoolID *uuid.UUID) error {
	if bodySchoolID == nil || *bodySchoolID == uuid.Nil {
		return nil
	}
	act, err := helperAuth.GetSchoolIDFromTokenPreferTeacher(c)
	if err != nil || act == uuid.Nil {
		return nil
	}
	if act != *bodySchoolID {
		return fiber.NewError(fiber.StatusForbidden, "school scope mismatch")
	}
	return nil
}

// --- PG error mapping (pgx/libpq) ---
func mapPGError(err error) (int, string) {
	// pgx
	var pgxErr *pgconn.PgError
	if errors.As(err, &pgxErr) {
		switch pgxErr.Code {
		case "23P01":
			return http.StatusConflict, "Bentrok jadwal: time range overlap (context: CSST/Hari/Jam)."
		case "23503":
			return http.StatusBadRequest, "Referensi tidak ditemukan (FK violation)."
		case "23505":
			return http.StatusConflict, "Data duplikat (unique violation)."
		default:
			return http.StatusInternalServerError, pgxErr.Message
		}
	}
	// lib/pq
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		code := string(pqErr.Code)
		switch code {
		case "23P01":
			return http.StatusConflict, "Bentrok jadwal: time range overlap (context: CSST/Hari/Jam)."
		case "23503":
			return http.StatusBadRequest, "Referensi tidak ditemukan (FK violation)."
		case "23505":
			return http.StatusConflict, "Data duplikat (unique violation)."
		default:
			return http.StatusInternalServerError, pqErr.Error()
		}
	}
	return http.StatusInternalServerError, err.Error()
}

func writePGError(c *fiber.Ctx, err error) error {
	code, msg := mapPGError(err)
	return helper.JsonError(c, code, msg)
}

// helper/parse.go
func ParseBoolLoose(s string) (bool, bool) {
	if s == "" {
		return false, false // not present
	}
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "t", "yes", "y", "on":
		return true, true
	case "0", "false", "f", "no", "n", "off":
		return false, true
	default:
		return false, false
	}
}

/* ==================================================================== */
/* Helper: ambil core CSST (tenant-safe) untuk snapshot & fallback      */
/* ==================================================================== */

// --- ganti helper csstCore & getCSSTCore ---

type csstCore struct {
	ID        uuid.UUID
	SchoolID  uuid.UUID
	Slug      *string
	SectionID *uuid.UUID

	// Di model baru kita sudah punya subject snapshot,
	// jadi cukup ambil dari situ (tanpa join ke class_subject_books)
	SubjectID *uuid.UUID

	TeacherID *uuid.UUID
	RoomID    *uuid.UUID
}

func getCSSTCore(tx *gorm.DB, schoolID, csstID uuid.UUID) (csstCore, error) {
	var r csstCore
	err := tx.
		Table("class_section_subject_teachers AS csst").
		Select(`
			csst.class_section_subject_teacher_id                       AS id,
			csst.class_section_subject_teacher_school_id                AS school_id,
			csst.class_section_subject_teacher_slug                     AS slug,
			csst.class_section_subject_teacher_class_section_id         AS section_id,

			-- pakai subject_id dari SNAPSHOT (tanpa join ke class_subjects/books)
			csst.class_section_subject_teacher_subject_id_snapshot      AS subject_id,

			csst.class_section_subject_teacher_school_teacher_id        AS teacher_id,
			csst.class_section_subject_teacher_class_room_id            AS room_id
		`).
		Where(`
			csst.class_section_subject_teacher_id = ?
			AND csst.class_section_subject_teacher_school_id = ?
			AND csst.class_section_subject_teacher_deleted_at IS NULL
		`, csstID, schoolID).
		Take(&r).Error
	if err != nil {
		return r, err
	}
	if r.SchoolID != schoolID {
		return r, fiber.NewError(fiber.StatusForbidden, "CSST milik school lain")
	}
	return r, nil
}

/*
=========================

	Create

=========================
*/
func (ctl *ClassScheduleController) Create(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	// 🔐 Guard role: hanya DKM + Teacher
	if !(helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return helper.JsonError(c, http.StatusForbidden, "Akses ditolak (hanya DKM/Guru yang diizinkan)")
	}

	// 1) school context: PRIORITAS token, fallback path/query/slug
	actSchoolID, err := resolveSchoolID(c)
	if err != nil {
		return err
	}

	// 2) Pastikan user memang DKM/Teacher di school ini
	if er := helperAuth.EnsureDKMOrTeacherSchool(c, actSchoolID); er != nil {
		return er
	}

	// 2) Body
	var req d.CreateClassScheduleRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("[ClassSchedule.Create] BodyParser error: %v", err)
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	// 3) Validasi payload
	if ctl.Validate != nil {
		if err := ctl.Validate.Struct(req); err != nil {
			log.Printf("[ClassSchedule.Create] Validation error: %v", err)
			return helper.JsonError(c, http.StatusBadRequest, err.Error())
		}
	}

	// 4) Build header
	header, err := req.ToModel(actSchoolID)
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	if header.ClassScheduleStartDate.After(header.ClassScheduleEndDate) {
		return helper.JsonError(c, http.StatusBadRequest, "start_date harus <= end_date")
	}

	// Flag generate
	doGen := true
	if q := c.Query("generate_sessions", ""); q != "" {
		if v, ok := ParseBoolLoose(q); ok {
			doGen = v
		} else {
			return helper.JsonError(c, http.StatusBadRequest, "Query generate_sessions harus boolean")
		}
	} else if req.GenerateSessions != nil {
		doGen = *req.GenerateSessions
	}

	var sessionsProvided []sessModel.ClassAttendanceSessionModel

	if err := ctl.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// (a) schedule
		if er := tx.Create(&header).Error; er != nil {
			return er
		}

		// (b) rules (opsional) — enrich CSST slug+snapshot
		if len(req.Rules) > 0 {
			ruleModels, er := req.RulesToModels(actSchoolID, header.ClassScheduleID)
			if er != nil {
				return er
			}
			for i := range ruleModels {
				csstID := ruleModels[i].ClassScheduleRuleCSSTID
				core, e := getCSSTCore(tx, actSchoolID, csstID)
				if e != nil {
					if errors.Is(e, gorm.ErrRecordNotFound) {
						return helper.JsonError(c, http.StatusBadRequest, "CSST tidak ditemukan / beda tenant")
					}
					var fe *fiber.Error
					if errors.As(e, &fe) {
						return helper.JsonError(c, fe.Code, fe.Message)
					}
					return e
				}
				ruleModels[i].ClassScheduleRuleCSSTSlugSnapshot = core.Slug
				ruleModels[i].ClassScheduleRuleCSSTSnapshot = datatypes.JSONMap{
					"school_id":             core.SchoolID.String(),
					"csst_id":               core.ID.String(),
					"slug":                  core.Slug,
					"section_id":            core.SectionID,
					"subject_id":            core.SubjectID,
					"teacher_id":            core.TeacherID,
					"room_id":               core.RoomID,
				}
			}
			if er := tx.Create(&ruleModels).Error; er != nil {
				return er
			}
		}

		// (c) sessions (opsional) — enrich snapshot CSST + fallback teacher/room
		if len(req.Sessions) > 0 {
			ms, er := req.SessionsToModels(
				actSchoolID,
				header.ClassScheduleID,
				header.ClassScheduleStartDate,
				header.ClassScheduleEndDate,
			)
			if er != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, er.Error())
			}

			for i := range ms {
				if ms[i].ClassAttendanceSessionCSSTID == nil {
					return helper.JsonError(c, fiber.StatusBadRequest, fmt.Sprintf("sessions[%d]: csst_id wajib", i))
				}
				core, e := getCSSTCore(tx, actSchoolID, *ms[i].ClassAttendanceSessionCSSTID)
				if e != nil {
					if errors.Is(e, gorm.ErrRecordNotFound) {
						return helper.JsonError(c, fiber.StatusBadRequest, fmt.Sprintf("sessions[%d]: CSST tidak ditemukan / beda tenant", i))
					}
					var fe *fiber.Error
					if errors.As(e, &fe) {
						return helper.JsonError(c, fe.Code, fe.Message)
					}
					return e
				}

				// Snapshot CSST (minimal namun cukup)
				ms[i].ClassAttendanceSessionCSSTSnapshot = datatypes.JSONMap{
					"school_id":             core.SchoolID.String(),
					"csst_id":               core.ID.String(),
					"slug":                  core.Slug,
					"section_id":            core.SectionID,
					"subject_id":            core.SubjectID,
					"teacher_id":            core.TeacherID,
					"room_id":               core.RoomID,
				}

				// Fallback override teacher/room jika payload kosong
				if ms[i].ClassAttendanceSessionTeacherID == nil && core.TeacherID != nil {
					v := *core.TeacherID
					ms[i].ClassAttendanceSessionTeacherID = &v
				}
				if ms[i].ClassAttendanceSessionClassRoomID == nil && core.RoomID != nil {
					v := *core.RoomID
					ms[i].ClassAttendanceSessionClassRoomID = &v
				}
			}

			if len(ms) > 0 {
				if er := tx.
					Clauses(clause.OnConflict{DoNothing: true}).
					Clauses(clause.Returning{}).
					Create(&ms).Error; er != nil {
					return er
				}
				sessionsProvided = ms // sudah berisi ID & timestamps dari DB
			}
		}

		return nil
	}); err != nil {
		if fiberErr, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fiberErr.Code, fiberErr.Message)
		}
		return writePGError(c, err)
	}

	// 6) Generate sessions dari rules (opsional)
	sessionsGenerated := 0
	var genErr error
	if doGen {
		var defCSST, defRoom, defTeacher *uuid.UUID
		if req.DefaultCSSTID != nil {
			v := *req.DefaultCSSTID
			defCSST = &v
		}
		if req.DefaultRoomID != nil {
			v := *req.DefaultRoomID
			defRoom = &v
		}
		if req.DefaultTeacherID != nil {
			v := *req.DefaultTeacherID
			defTeacher = &v
		}

		// fallback dari sessions payload (kalau ada)
		for _, s := range req.Sessions {
			if s.CSSTID != nil && defCSST == nil {
				v := *s.CSSTID
				defCSST = &v
			}
			if s.ClassRoomID != nil && defRoom == nil {
				v := *s.ClassRoomID
				defRoom = &v
			}
		}

		gen := svc.Generator{DB: ctl.DB}
		sessionsGenerated, genErr = gen.GenerateSessionsForScheduleWithOpts(
			c.Context(),
			header.ClassScheduleID.String(),
			&svc.GenerateOptions{
				TZName:                  "Asia/Jakarta",
				DefaultCSSTID:           defCSST,
				DefaultRoomID:           defRoom,
				DefaultTeacherID:        defTeacher,
				DefaultAttendanceStatus: "open",
				BatchSize:               500,
			},
		)
		if genErr != nil {
			log.Printf("[ClassSchedule.Create] Generate error: %v", genErr)
		}
	}

	// 7) Response
	resp := fiber.Map{
		"schedule":           d.FromModel(header),
		"sessions_provided":  len(sessionsProvided),
		"sessions_generated": sessionsGenerated,
		"generated":          doGen,
	}
	if genErr != nil {
		resp["generation_warning"] = genErr.Error()
	}
	return helper.JsonCreated(c, "Schedule created", resp)
}

/* =========================
   Patch (Partial) — pointer-based DTO
   ========================= */

func (ctl *ClassScheduleController) Patch(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	// 🔐 Guard role: hanya DKM + Teacher (global)
	if !(helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return helper.JsonError(c, http.StatusForbidden, "Akses ditolak (hanya DKM/Guru yang diizinkan)")
	}

	// 1) school context: PRIORITAS token, fallback path/query/slug
	actSchoolID, err := resolveSchoolID(c)
	if err != nil {
		return err
	}

	// 2) Pastikan user memang DKM/Teacher di school ini
	if er := helperAuth.EnsureDKMOrTeacherSchool(c, actSchoolID); er != nil {
		return er
	}

	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	var existing m.ClassScheduleModel
	if err := ctl.DB.
		Where("class_schedule_id = ? AND class_schedule_deleted_at IS NULL", id).
		First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "schedule not found")
		}
		return writePGError(c, err)
	}

	// DTO
	var req d.UpdateClassScheduleRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	if ctl.Validate != nil {
		if err := ctl.Validate.Struct(req); err != nil {
			return helper.JsonError(c, http.StatusBadRequest, err.Error())
		}
	}
	req.Apply(&existing)

	// School scope check (allow via DKM context)
	if err := enforceSchoolScopeAuth(c, &existing.ClassScheduleSchoolID); err != nil {
		if mc, er := helperAuth.ResolveSchoolContext(c); er == nil && (mc.ID != uuid.Nil || strings.TrimSpace(mc.Slug) != "") {
			if idOK, er2 := helperAuth.EnsureSchoolAccessDKM(c, mc); er2 == nil && idOK == existing.ClassScheduleSchoolID {
				// allowed
			} else {
				return helper.JsonError(c, http.StatusForbidden, "school scope mismatch")
			}
		} else {
			return helper.JsonError(c, http.StatusForbidden, "school scope mismatch")
		}
	}

	if err := ctl.DB.Save(&existing).Error; err != nil {
		return writePGError(c, err)
	}

	return helper.JsonUpdated(c, "Schedule updated", d.FromModel(existing))
}

func (ctl *ClassScheduleController) Delete(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	// 🔐 Guard role: hanya DKM + Teacher
	if !(helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return helper.JsonError(c, http.StatusForbidden, "Akses ditolak (hanya DKM/Guru yang diizinkan)")
	}

	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	var existing m.ClassScheduleModel
	if err := ctl.DB.
		Where("class_schedule_id = ? AND class_schedule_deleted_at IS NULL", id).
		First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "schedule not found")
		}
		return writePGError(c, err)
	}

	// School scope check (allow via DKM context)
	if err := enforceSchoolScopeAuth(c, &existing.ClassScheduleSchoolID); err != nil {
		if mc, er := helperAuth.ResolveSchoolContext(c); er == nil && (mc.ID != uuid.Nil || strings.TrimSpace(mc.Slug) != "") {
			if idOK, er2 := helperAuth.EnsureSchoolAccessDKM(c, mc); er2 == nil && idOK == existing.ClassScheduleSchoolID {
				// allowed
			} else {
				return helper.JsonError(c, http.StatusForbidden, "school scope mismatch")
			}
		} else {
			return helper.JsonError(c, http.StatusForbidden, "school scope mismatch")
		}
	}

	// 🔒 GUARD 1: masih ada class_schedule_rules yang pakai schedule ini?
	var ruleCount int64
	if err := ctl.DB.
		Model(&m.ClassScheduleRuleModel{}).
		Where(`
			class_schedule_rule_schedule_id = ?
			AND class_schedule_rule_deleted_at IS NULL
		`, existing.ClassScheduleID).
		Count(&ruleCount).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, "Gagal mengecek relasi rules")
	}

	// 🔒 GUARD 2: masih ada class_attendance_sessions yang pakai schedule ini?
	var sessCount int64
	if err := ctl.DB.
		Model(&sessModel.ClassAttendanceSessionModel{}).
		Where(`
			class_attendance_session_schedule_id = ?
			AND class_attendance_session_deleted_at IS NULL
		`, existing.ClassScheduleID).
		Count(&sessCount).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, "Gagal mengecek relasi sesi")
	}

	if ruleCount > 0 || sessCount > 0 {
		return helper.JsonError(
			c,
			http.StatusBadRequest,
			"Tidak dapat menghapus jadwal karena masih ada rule atau sesi absensi yang terhubung. "+
				"Mohon hapus / sesuaikan rule dan sesi terkait terlebih dahulu.",
		)
	}

	// ✅ Aman: tidak ada rule/sesi terhubung → lanjut soft delete
	if err := ctl.DB.Delete(&existing).Error; err != nil {
		return writePGError(c, err)
	}

	return helper.JsonDeleted(c, "Schedule deleted", d.FromModel(existing))
}
