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
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	d "masjidku_backend/internals/features/school/academics/schedules/dto"
	m "masjidku_backend/internals/features/school/academics/schedules/model"
	sessModel "masjidku_backend/internals/features/school/classes/class_attendance_sessions/model"
	svc "masjidku_backend/internals/features/school/academics/schedules/services" // ⬅️ generator sessions
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

// --- PG error mapping ---

type pgSQLErr interface {
	SQLState() string
	Error() string
}

func mapPGError(err error) (int, string) {
	// 23P01 = exclusion_violation
	// 23503 = foreign_key_violation
	// 23505 = unique_violation
	var pgErr pgSQLErr
	if errors.As(err, &pgErr) {
		switch pgErr.SQLState() {
		case "23P01":
			return http.StatusConflict, "Bentrok jadwal: time range overlap (context: CSST/Hari/Jam)."
		case "23503":
			return http.StatusBadRequest, "Referensi tidak ditemukan (FK violation)."
		case "23505":
			return http.StatusConflict, "Data duplikat (unique violation)."
		}
	}
	return http.StatusInternalServerError, err.Error()
}

func writePGError(c *fiber.Ctx, err error) error {
	code, msg := mapPGError(err)
	return helper.JsonError(c, code, msg)
}

/*
========================= Create =========================
*/

func (ctl *ClassScheduleController) Create(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	// --- guard
	if !(helperAuth.IsOwner(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return helper.JsonError(c, http.StatusForbidden, "Akses ditolak")
	}

	// --- body
	var req d.CreateClassScheduleRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("[ClassSchedule.Create] BodyParser error: %v", err)
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	// --- masjid scope
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

	// --- validate
	if ctl.Validate != nil {
		if err := ctl.Validate.Struct(req); err != nil {
			log.Printf("[ClassSchedule.Create] Validation error: %v", err)
			return helper.JsonError(c, http.StatusBadRequest, err.Error())
		}
	}

	// --- build schedule
	model := req.ToModel(actMasjidID)

	var sessionsProvided []sessModel.ClassAttendanceSessionModel

	// --- TX: create schedule + rules + sessions (manual)
	if err := ctl.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// 1) schedule
		if er := tx.Create(&model).Error; er != nil {
			log.Printf("[ClassSchedule.Create] DB.Create(schedule) error: %v", er)
			return er
		}

		// 2) rules (opsional)
		if len(req.Rules) > 0 {
			ruleModels, er := req.RulesToModels(actMasjidID, model.ClassScheduleID)
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

		// 3) sessions (opsional, manual dari payload)
		if len(req.Sessions) > 0 {
			ms, er := req.SessionsToModels(
				actMasjidID,
				model.ClassScheduleID,
				model.ClassScheduleStartDate,
				model.ClassScheduleEndDate,
			)
			if er != nil {
				log.Printf("[ClassSchedule.Create] SessionsToModels error: %v", er)
				return helper.JsonError(c, http.StatusBadRequest, er.Error())
			}

			if len(ms) > 0 {
				// idempotent insert (butuh unique idx (schedule_id, starts_at) partial alive)
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
		return writePGError(c, err)
	}

	// --- 4) SELALU generate dari rules (tanpa query param)
	// Ambil default assignment dari payload (pakai entri pertama yang ada nilainya)
	var defCSST, defRoom, defTeacher *uuid.UUID
	for _, s := range req.Sessions {
		if s.CSSTID != nil && defCSST == nil {
			v := *s.CSSTID
			defCSST = &v
		}
		if s.ClassRoomID != nil && defRoom == nil {
			v := *s.ClassRoomID
			defRoom = &v
		}
		// NOTE: kalau nanti DTO punya TeacherID, isi juga di sini.
	}

	gen := svc.Generator{DB: ctl.DB}
	sessionsGenerated, genErr := gen.GenerateSessionsForScheduleWithOpts(
		c.Context(),
		model.ClassScheduleID.String(),
		&svc.GenerateOptions{
			TZName:                  "Asia/Jakarta", // TODO: ambil dari profil masjid kalau ada
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

	// --- response
	resp := fiber.Map{
		"schedule":           d.FromModel(model),
		"sessions_provided":  len(sessionsProvided),
		"sessions_generated": sessionsGenerated,
		"generated":          true,
	}
	if genErr != nil {
		resp["generation_warning"] = genErr.Error()
	}
	return helper.JsonCreated(c, "Schedule created", resp)
}

/* =========================
   Patch (Partial) — gunakan DTO Update yang pointer-based
   ========================= */

func (ctl *ClassScheduleController) Patch(c *fiber.Ctx) error {
	// 🔁 masjid-context: siapkan DB untuk resolver
	c.Locals("DB", ctl.DB)

	// 🔐 Admin/DKM/Teacher
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

	// Gunakan UpdateClassScheduleRequest untuk PATCH (semua field pointer)
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

	// 🔁 masjid-context: izinkan DKM sesuai context meski token scope mismatch
	if err := enforceMasjidScopeAuth(c, &existing.ClassScheduleMasjidID); err != nil {
		if mc, er := helperAuth.ResolveMasjidContext(c); er == nil && (mc.ID != uuid.Nil || strings.TrimSpace(mc.Slug) != "") {
			if idOK, er2 := helperAuth.EnsureMasjidAccessDKM(c, mc); er2 == nil && idOK == existing.ClassScheduleMasjidID {
				// ✅ allow via DKM context
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
	// 🔁 masjid-context: siapkan DB untuk resolver
	c.Locals("DB", ctl.DB)

	// 🔐 Admin/DKM/Teacher
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

	// 🔁 masjid-context: izinkan DKM sesuai context meski token scope mismatch
	if err := enforceMasjidScopeAuth(c, &existing.ClassScheduleMasjidID); err != nil {
		if mc, er := helperAuth.ResolveMasjidContext(c); er == nil && (mc.ID != uuid.Nil || strings.TrimSpace(mc.Slug) != "") {
			if idOK, er2 := helperAuth.EnsureMasjidAccessDKM(c, mc); er2 == nil && idOK == existing.ClassScheduleMasjidID {
				// ✅ allow via DKM context
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
