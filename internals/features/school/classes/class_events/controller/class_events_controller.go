// file: internals/features/school/sessions/events/controller/class_events_controller.go
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

	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	d "madinahsalam_backend/internals/features/school/classes/class_events/dto"
	m "madinahsalam_backend/internals/features/school/classes/class_events/model"
)

/* =========================
   Controller & Constructor
   ========================= */

type ClassEventsController struct {
	DB       *gorm.DB
	Validate *validator.Validate
}

func NewClassEvents(db *gorm.DB, v *validator.Validate) *ClassEventsController {
	return &ClassEventsController{DB: db, Validate: v}
}

/* =========================
   Small helpers
   ========================= */

func parseUUIDParam(c *fiber.Ctx, name string) (uuid.UUID, error) {
	idStr := strings.TrimSpace(c.Params(name))
	if idStr == "" {
		return uuid.Nil, fmt.Errorf("%s is required", name)
	}
	return uuid.Parse(idStr)
}

// parse YYYY-MM-DD → time.Time (UTC midnight)
func parseDateYYYYMMDD(s string) (time.Time, bool) {
	t, err := time.Parse("2006-01-02", strings.TrimSpace(s))
	if err != nil {
		return time.Time{}, false
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), true
}

// --- PG error mapping ---
type pgSQLErr interface {
	SQLState() string
	Error() string
}

func mapPGError(err error) (int, string) {
	// 23505 unique_violation, 23503 foreign_key_violation
	var pgErr pgSQLErr
	if errors.As(err, &pgErr) {
		switch pgErr.SQLState() {
		case "23505":
			return http.StatusConflict, "Data duplikat (unique violation)."
		case "23503":
			return http.StatusBadRequest, "Referensi tidak ditemukan (FK violation)."
		}
	}
	return http.StatusInternalServerError, err.Error()
}

func writePGError(c *fiber.Ctx, err error) error {
	code, msg := mapPGError(err)
	return helper.JsonError(c, code, msg)
}

/* =========================
   Create  (OWNER or DKM/Admin School)
   Context via resolver (path/header/query/host/token)
   ========================= */

func (ctl *ClassEventsController) Create(c *fiber.Ctx) error {
	// sediakan DB untuk resolver slug→id
	c.Locals("DB", ctl.DB)

	// resolve school context
	mc, err := helperAuth.ResolveSchoolContext(c)
	if err != nil {
		return err
	}

	// tentukan school aktif
	var actSchoolID uuid.UUID
	if helperAuth.IsOwner(c) {
		// owner: tidak perlu cek role, tapi tetap perlu id school
		if mc.ID != uuid.Nil {
			actSchoolID = mc.ID
		} else if strings.TrimSpace(mc.Slug) != "" {
			id, er := helperAuth.GetSchoolIDBySlug(c, mc.Slug)
			if er != nil {
				return fiber.NewError(fiber.StatusNotFound, "School (slug) tidak ditemukan")
			}
			actSchoolID = id
		} else {
			return helperAuth.ErrSchoolContextMissing
		}
	} else {
		// non-owner: wajib DKM/Admin
		id, er := helperAuth.EnsureSchoolAccessDKM(c, mc)
		if er != nil {
			return er
		}
		actSchoolID = id
	}

	var req d.CreateClassEventRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	if ctl.Validate != nil {
		if err := ctl.Validate.Struct(req); err != nil {
			return helper.JsonError(c, http.StatusBadRequest, err.Error())
		}
	}

	model, err := req.ToModel(actSchoolID)
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	if err := ctl.DB.WithContext(c.Context()).Create(&model).Error; err != nil {
		return writePGError(c, err)
	}

	return helper.JsonCreated(c, "Class event created", d.FromModelClassEvent(model))
}

/* =========================
   Patch  (OWNER or DKM/Admin School)
   ========================= */

func (ctl *ClassEventsController) Patch(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	mc, err := helperAuth.ResolveSchoolContext(c)
	if err != nil {
		return err
	}

	var actSchoolID uuid.UUID
	if helperAuth.IsOwner(c) {
		if mc.ID != uuid.Nil {
			actSchoolID = mc.ID
		} else if strings.TrimSpace(mc.Slug) != "" {
			id, er := helperAuth.GetSchoolIDBySlug(c, mc.Slug)
			if er != nil {
				return fiber.NewError(fiber.StatusNotFound, "School (slug) tidak ditemukan")
			}
			actSchoolID = id
		} else {
			return helperAuth.ErrSchoolContextMissing
		}
	} else {
		id, er := helperAuth.EnsureSchoolAccessDKM(c, mc)
		if er != nil {
			return er
		}
		actSchoolID = id
	}

	eventID, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	var existing m.ClassEventModel
	if err := ctl.DB.
		Where("class_event_id = ? AND class_event_school_id = ? AND class_event_deleted_at IS NULL", eventID, actSchoolID).
		First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "class event not found")
		}
		return writePGError(c, err)
	}

	var req d.PatchClassEventRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	if err := req.Apply(&existing); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	if err := ctl.DB.WithContext(c.Context()).Save(&existing).Error; err != nil {
		return writePGError(c, err)
	}

	return helper.JsonUpdated(c, "Class event updated", d.FromModelClassEvent(existing))
}

/* =========================
   Delete (soft)  (OWNER or DKM/Admin School)
   ========================= */

func (ctl *ClassEventsController) Delete(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	mc, err := helperAuth.ResolveSchoolContext(c)
	if err != nil {
		return err
	}

	var actSchoolID uuid.UUID
	if helperAuth.IsOwner(c) {
		if mc.ID != uuid.Nil {
			actSchoolID = mc.ID
		} else if strings.TrimSpace(mc.Slug) != "" {
			id, er := helperAuth.GetSchoolIDBySlug(c, mc.Slug)
			if er != nil {
				return fiber.NewError(fiber.StatusNotFound, "School (slug) tidak ditemukan")
			}
			actSchoolID = id
		} else {
			return helperAuth.ErrSchoolContextMissing
		}
	} else {
		id, er := helperAuth.EnsureSchoolAccessDKM(c, mc)
		if er != nil {
			return er
		}
		actSchoolID = id
	}

	eventID, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	var existing m.ClassEventModel
	if err := ctl.DB.
		Where("class_event_id = ? AND class_event_school_id = ? AND class_event_deleted_at IS NULL", eventID, actSchoolID).
		First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "class event not found")
		}
		return writePGError(c, err)
	}

	if err := ctl.DB.WithContext(c.Context()).Delete(&existing).Error; err != nil {
		return writePGError(c, err)
	}

	return helper.JsonOK(c, "Class event deleted", fiber.Map{"class_event_id": eventID})
}
