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
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	sessModel "masjidku_backend/internals/features/school/classes/class_attendance_sessions/model"
	d "masjidku_backend/internals/features/school/classes/class_schedules/dto"
	m "masjidku_backend/internals/features/school/classes/class_schedules/model"
	svc "masjidku_backend/internals/features/school/classes/class_schedules/services"
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

// Ambil masjid_id aktif dari token (teacher/admin/dkm) dan pastikan konsisten dengan body.
// Jika token tidak punya scope masjid, dilepas (public).
func enforceMasjidScopeAuth(c *fiber.Ctx, bodyMasjidID *uuid.UUID) error {
	if bodyMasjidID == nil || *bodyMasjidID == uuid.Nil {
		return nil
	}
	act, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || act == uuid.Nil {
		return nil
	}
	if act != *bodyMasjidID {
		return fiber.NewError(fiber.StatusForbidden, "masjid scope mismatch")
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

/* =========================
   Create (schedule + optional rules & sessions)
   ========================= */

func (ctl *ClassScheduleController) Create(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	// Guard role
	if !(helperAuth.IsOwner(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return helper.JsonError(c, http.StatusForbidden, "Akses ditolak")
	}

	// Body
	var req d.CreateClassScheduleRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("[ClassSchedule.Create] BodyParser error: %v", err)
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	// Masjid scope
	var actMasjidID uuid.UUID
	if mc, err := helperAuth.ResolveMasjidContext(c); err == nil && (mc.ID != uuid.Nil || strings.TrimSpace(mc.Slug) != "") {
		id, er := helperAuth.EnsureMasjidAccessDKM(c, mc)
		if er != nil {
			log.Printf("[ClassSchedule.Create] EnsureMasjidAccessDKM error: %v", er)
			return er
		}
		actMasjidID = id
	} else {
		id, er := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
		if er != nil || id == uuid.Nil {
			log.Printf("[ClassSchedule.Create] Masjid scope tidak ditemukan: %v", er)
			return helper.JsonError(c, http.StatusUnauthorized, "Masjid scope tidak ditemukan di token")
		}
		actMasjidID = id
	}

	// Validate
	if ctl.Validate != nil {
		if err := ctl.Validate.Struct(req); err != nil {
			log.Printf("[ClassSchedule.Create] Validation error: %v", err)
			return helper.JsonError(c, http.StatusBadRequest, err.Error())
		}
	}

	// Build header
	header := req.ToModel(actMasjidID)
	if header.ClassScheduleStartDate.After(header.ClassScheduleEndDate) {
		return helper.JsonError(c, http.StatusBadRequest, "start_date harus <= end_date")
	}

	// ===== Tentukan apakah ingin auto-generate sessions dari rules =====
	// Priority: query param > body > default(true)
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

	// TX: create schedule + rules + sessions(payload)
	if err := ctl.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// 1) schedule
		if er := tx.Create(&header).Error; er != nil {
			log.Printf("[ClassSchedule.Create] DB.Create(schedule) error: %v", er)
			return er
		}

		// 2) rules (opsional)
		if len(req.Rules) > 0 {
			ruleModels, er := req.RulesToModels(actMasjidID, header.ClassScheduleID)
			if er != nil {
				log.Printf("[ClassSchedule.Create] RulesToModels error: %v", er)
				return er
			}
			if len(ruleModels) > 0 {
				if er := tx.Create(&ruleModels).Error; er != nil {
					log.Printf("[ClassSchedule.Create] DB.Create(rules) error: %v", er)
					return er
				}
			}
		}

		// 3) sessions (opsional, langsung dari payload)
		if len(req.Sessions) > 0 {
			ms, er := req.SessionsToModels(
				actMasjidID,
				header.ClassScheduleID,
				header.ClassScheduleStartDate,
				header.ClassScheduleEndDate,
			)
			if er != nil {
				log.Printf("[ClassSchedule.Create] SessionsToModels error: %v", er)
				// balikan 400 rapi
				return helper.JsonError(c, http.StatusBadRequest, er.Error())
			}
			if len(ms) > 0 {
				if er := tx.
					Clauses(clause.OnConflict{DoNothing: true}).
					Create(&ms).Error; er != nil {
					log.Printf("[ClassSchedule.Create] DB.Create(sessions) error: %v", er)
					return er
				}
				sessionsProvided = ms
			}
		}
		return nil
	}); err != nil {
		if fiberErr, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fiberErr.Code, fiberErr.Message)
		}
		return writePGError(c, err)
	}

	// 4) Generate sessions dari rules → HANYA jika doGen == true
	sessionsGenerated := 0
	var genErr error
	if doGen {
		var defCSST, defRoom, defTeacher *uuid.UUID

		// (A) Ambil dari payload default_* (opsional)
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

		// (B) Fallback dari sessions yang ikut dikirim (kalau ada)
		for _, s := range req.Sessions {
			if s.CSSTID != nil && defCSST == nil {
				v := *s.CSSTID
				defCSST = &v
			}
			if s.ClassRoomID != nil && defRoom == nil {
				v := *s.ClassRoomID
				defRoom = &v
			}
			// kalau SessionCreateDTO punya TeacherID, bisa aktifkan:
			// if s.TeacherID != nil && defTeacher == nil {
			//    v := *s.TeacherID
			//    defTeacher = &v
			// }
		}

		gen := svc.Generator{DB: ctl.DB}
		sessionsGenerated, genErr = gen.GenerateSessionsForScheduleWithOpts(
			c.Context(),
			header.ClassScheduleID.String(),
			&svc.GenerateOptions{
				TZName:                  "Asia/Jakarta", // TODO: tarik dari profil masjid
				DefaultCSSTID:           defCSST,
				DefaultRoomID:           defRoom,
				DefaultTeacherID:        defTeacher,
				DefaultAttendanceStatus: "open",
				BatchSize:               500,
			},
		)
		if genErr != nil {
			log.Printf("[ClassSchedule.Create] GenerateSessionsForSchedule error: %v", genErr)
		}
	}

	// Response
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

	// Guard role
	if !(helperAuth.IsOwner(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return helper.JsonError(c, http.StatusForbidden, "Akses ditolak")
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

	// Masjid scope check (allow via DKM context)
	if err := enforceMasjidScopeAuth(c, &existing.ClassScheduleMasjidID); err != nil {
		if mc, er := helperAuth.ResolveMasjidContext(c); er == nil && (mc.ID != uuid.Nil || strings.TrimSpace(mc.Slug) != "") {
			if idOK, er2 := helperAuth.EnsureMasjidAccessDKM(c, mc); er2 == nil && idOK == existing.ClassScheduleMasjidID {
				// allowed
			} else {
				return helper.JsonError(c, http.StatusForbidden, "masjid scope mismatch")
			}
		} else {
			return helper.JsonError(c, http.StatusForbidden, "masjid scope mismatch")
		}
	}

	if err := ctl.DB.Save(&existing).Error; err != nil {
		return writePGError(c, err)
	}

	return helper.JsonUpdated(c, "Schedule updated", d.FromModel(existing))
}

/* =========================
   Soft Delete
   ========================= */

func (ctl *ClassScheduleController) Delete(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	// Guard role
	if !(helperAuth.IsOwner(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return helper.JsonError(c, http.StatusForbidden, "Akses ditolak")
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

	// Masjid scope check (allow via DKM context)
	if err := enforceMasjidScopeAuth(c, &existing.ClassScheduleMasjidID); err != nil {
		if mc, er := helperAuth.ResolveMasjidContext(c); er == nil && (mc.ID != uuid.Nil || strings.TrimSpace(mc.Slug) != "") {
			if idOK, er2 := helperAuth.EnsureMasjidAccessDKM(c, mc); er2 == nil && idOK == existing.ClassScheduleMasjidID {
				// allowed
			} else {
				return helper.JsonError(c, http.StatusForbidden, "masjid scope mismatch")
			}
		} else {
			return helper.JsonError(c, http.StatusForbidden, "masjid scope mismatch")
		}
	}

	// GORM soft delete → set class_schedule_deleted_at
	if err := ctl.DB.Delete(&existing).Error; err != nil {
		return writePGError(c, err)
	}

	return helper.JsonDeleted(c, "Schedule deleted", d.FromModel(existing))
}
