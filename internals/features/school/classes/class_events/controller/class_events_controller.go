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

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	d "masjidku_backend/internals/features/school/classes/class_events/dto"
	m "masjidku_backend/internals/features/school/classes/class_events/model"
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
   Create  (OWNER or DKM/Admin Masjid)
   Context via resolver (path/header/query/host/token)
   ========================= */

func (ctl *ClassEventsController) Create(c *fiber.Ctx) error {
	// sediakan DB untuk resolver slug→id
	c.Locals("DB", ctl.DB)

	// resolve masjid context
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}

	// tentukan masjid aktif
	var actMasjidID uuid.UUID
	if helperAuth.IsOwner(c) {
		// owner: tidak perlu cek role, tapi tetap perlu id masjid
		if mc.ID != uuid.Nil {
			actMasjidID = mc.ID
		} else if strings.TrimSpace(mc.Slug) != "" {
			id, er := helperAuth.GetMasjidIDBySlug(c, mc.Slug)
			if er != nil {
				return fiber.NewError(fiber.StatusNotFound, "Masjid (slug) tidak ditemukan")
			}
			actMasjidID = id
		} else {
			return helperAuth.ErrMasjidContextMissing
		}
	} else {
		// non-owner: wajib DKM/Admin
		id, er := helperAuth.EnsureMasjidAccessDKM(c, mc)
		if er != nil {
			return er
		}
		actMasjidID = id
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

	model, err := req.ToModel(actMasjidID)
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	if err := ctl.DB.WithContext(c.Context()).Create(&model).Error; err != nil {
		return writePGError(c, err)
	}

	return helper.JsonCreated(c, "Class event created", d.FromModelClassEvent(model))
}

/* =========================
   Patch  (OWNER or DKM/Admin Masjid)
   ========================= */

func (ctl *ClassEventsController) Patch(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}

	var actMasjidID uuid.UUID
	if helperAuth.IsOwner(c) {
		if mc.ID != uuid.Nil {
			actMasjidID = mc.ID
		} else if strings.TrimSpace(mc.Slug) != "" {
			id, er := helperAuth.GetMasjidIDBySlug(c, mc.Slug)
			if er != nil {
				return fiber.NewError(fiber.StatusNotFound, "Masjid (slug) tidak ditemukan")
			}
			actMasjidID = id
		} else {
			return helperAuth.ErrMasjidContextMissing
		}
	} else {
		id, er := helperAuth.EnsureMasjidAccessDKM(c, mc)
		if er != nil {
			return er
		}
		actMasjidID = id
	}

	eventID, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	var existing m.ClassEventModel
	if err := ctl.DB.
		Where("class_event_id = ? AND class_event_masjid_id = ? AND class_event_deleted_at IS NULL", eventID, actMasjidID).
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
   Delete (soft)  (OWNER or DKM/Admin Masjid)
   ========================= */

func (ctl *ClassEventsController) Delete(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}

	var actMasjidID uuid.UUID
	if helperAuth.IsOwner(c) {
		if mc.ID != uuid.Nil {
			actMasjidID = mc.ID
		} else if strings.TrimSpace(mc.Slug) != "" {
			id, er := helperAuth.GetMasjidIDBySlug(c, mc.Slug)
			if er != nil {
				return fiber.NewError(fiber.StatusNotFound, "Masjid (slug) tidak ditemukan")
			}
			actMasjidID = id
		} else {
			return helperAuth.ErrMasjidContextMissing
		}
	} else {
		id, er := helperAuth.EnsureMasjidAccessDKM(c, mc)
		if er != nil {
			return er
		}
		actMasjidID = id
	}

	eventID, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	var existing m.ClassEventModel
	if err := ctl.DB.
		Where("class_event_id = ? AND class_event_masjid_id = ? AND class_event_deleted_at IS NULL", eventID, actMasjidID).
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
