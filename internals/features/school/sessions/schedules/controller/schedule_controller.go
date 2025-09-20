// file: internals/features/school/class_schedules/controller/class_schedule_controller.go
package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	d "masjidku_backend/internals/features/school/sessions/schedules/dto"
	m "masjidku_backend/internals/features/school/sessions/schedules/model"
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

func parseTimeOfDayParam(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty time")
	}
	if t, err := time.Parse("15:04", s); err == nil {
		return t, nil
	}
	if t, err := time.Parse("15:04:05", s); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("invalid time format (want HH:mm or HH:mm:ss)")
}

/*
========================= Create =========================
*/
func (ctl *ClassScheduleController) Create(c *fiber.Ctx) error {
	// 🔁 masjid-context: siapkan DB untuk resolver
	c.Locals("DB", ctl.DB)

	// 🔐 Admin/DKM/Teacher
	if !(helperAuth.IsOwner(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return helper.JsonError(c, http.StatusForbidden, "Akses ditolak")
	}

	var req d.CreateClassScheduleRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	// 🔁 masjid-context: coba resolve dari path/header/query/host; jika ada → wajib DKM di masjid tsb
	var actMasjidID uuid.UUID
	if mc, err := helperAuth.ResolveMasjidContext(c); err == nil && (mc.ID != uuid.Nil || strings.TrimSpace(mc.Slug) != "") {
		id, er := helperAuth.EnsureMasjidAccessDKM(c, mc)
		if er != nil {
			return er
		}
		actMasjidID = id
	} else {
		// fallback ke token (existing behavior)
		id, er := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
		if er != nil || id == uuid.Nil {
			return helper.JsonError(c, http.StatusUnauthorized, "Masjid scope tidak ditemukan di token")
		}
		actMasjidID = id
	}

	// Override body
	req.ClassSchedulesMasjidID = actMasjidID.String()

	// Validasi setelah di-inject
	if err := req.Validate(ctl.Validate); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	var model m.ClassScheduleModel
	if err := req.ApplyToModel(&model); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	if err := ctl.DB.Create(&model).Error; err != nil {
		return writePGError(c, err)
	}

	return helper.JsonCreated(c, "Schedule created", d.NewClassScheduleResponse(&model))
}

/* =========================
   Update (PUT)
   ========================= */

func (ctl *ClassScheduleController) Update(c *fiber.Ctx) error {
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
		Where("class_schedule_id = ? AND class_schedules_deleted_at IS NULL", id).
		First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "schedule not found")
		}
		return writePGError(c, err)
	}

	var req d.UpdateClassScheduleRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	if err := req.Validate(ctl.Validate); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	if err := req.ApplyToModel(&existing); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
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

	if err := ctl.DB.Save(&existing).Error; err != nil {
		return writePGError(c, err)
	}

	return helper.JsonUpdated(c, "Schedule updated", d.NewClassScheduleResponse(&existing))
}

/* =========================
   Patch (Partial)
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
		Where("class_schedule_id = ? AND class_schedules_deleted_at IS NULL", id).
		First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "schedule not found")
		}
		return writePGError(c, err)
	}

	var req d.PatchClassScheduleRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	if err := req.Validate(ctl.Validate); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	if err := req.ApplyPatch(&existing); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
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

	if err := ctl.DB.Save(&existing).Error; err != nil {
		return writePGError(c, err)
	}

	return helper.JsonUpdated(c, "Schedule updated", d.NewClassScheduleResponse(&existing))
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
		Where("class_schedule_id = ? AND class_schedules_deleted_at IS NULL", id).
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

	// GORM soft delete → set class_schedules_deleted_at
	if err := ctl.DB.Delete(&existing).Error; err != nil {
		return writePGError(c, err)
	}

	return helper.JsonDeleted(c, "Schedule deleted", d.NewClassScheduleResponse(&existing))
}
