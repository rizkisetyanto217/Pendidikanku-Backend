// file: internals/features/school/events/themes/controller/class_event_theme_controller.go
package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	dto "madinahsalam_backend/internals/features/school/classes/class_events/dto"
	model "madinahsalam_backend/internals/features/school/classes/class_events/model"
)

/* =========================
   Controller
   ========================= */

type ClassEventThemeController struct {
	DB        *gorm.DB
	Validator *validator.Validate
}

func NewClassEventThemeController(db *gorm.DB) *ClassEventThemeController {
	return &ClassEventThemeController{
		DB:        db,
		Validator: validator.New(),
	}
}

/* =========================
   Small helpers
   ========================= */

func (ctl *ClassEventThemeController) getID(c *fiber.Ctx) (uuid.UUID, error) {
	param := strings.TrimSpace(c.Params("id"))
	if param == "" {
		return uuid.Nil, errors.New("missing id")
	}
	id, err := uuid.Parse(param)
	if err != nil {
		return uuid.Nil, errors.New("invalid id")
	}
	return id, nil
}

func queryIntDefault(c *fiber.Ctx, key string, def int) int {
	v := strings.TrimSpace(c.Query(key))
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return i
}

// isDuplicateKey: cek pelanggaran unique Postgres (SQLSTATE 23505).
func isDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key") || strings.Contains(msg, "unique constraint") || strings.Contains(msg, "23505")
}

// resolveSchoolAndEnsureDKM: gunakan helper resolver & guard
func (ctl *ClassEventThemeController) resolveSchoolAndEnsureDKM(c *fiber.Ctx) (uuid.UUID, error) {
	mc, err := helperAuth.ResolveSchoolContext(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			_ = helper.JsonError(c, fe.Code, fe.Message)
			return uuid.Nil, err
		}
		_ = helper.JsonError(c, http.StatusBadRequest, err.Error())
		return uuid.Nil, err
	}
	schoolID, err := helperAuth.EnsureSchoolAccessDKM(c, mc)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			_ = helper.JsonError(c, fe.Code, fe.Message)
			return uuid.Nil, err
		}
		_ = helper.JsonError(c, http.StatusForbidden, err.Error())
		return uuid.Nil, err
	}
	return schoolID, nil
}

/*
=========================================================

	CREATE
	POST /api/a/:school_id/events/themes
	Body: JSON CreateClassEventThemeRequest

=========================================================
*/
func (ctl *ClassEventThemeController) Create(c *fiber.Ctx) error {
	schoolID, err := ctl.resolveSchoolAndEnsureDKM(c)
	if err != nil {
		return nil
	}

	var req dto.CreateClassEventThemeRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	req.Normalize()
	if err := req.Validate(ctl.Validator); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	m := req.ToModel(schoolID)

	if err := ctl.DB.Create(m).Error; err != nil {
		if isDuplicateKey(err) {
			return helper.JsonError(c, http.StatusConflict, "theme code already exists for this school")
		}
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	return helper.JsonCreated(c, "Created", dto.FromModel(m))
}

/*
=========================================================

	PATCH
	PATCH /api/a/:school_id/events/themes/:id
	Body: JSON PatchClassEventThemeRequest (tri-state)

=========================================================
*/
func (ctl *ClassEventThemeController) Patch(c *fiber.Ctx) error {
	schoolID, err := ctl.resolveSchoolAndEnsureDKM(c)
	if err != nil {
		return nil
	}

	id, err := ctl.getID(c)
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	var req dto.PatchClassEventThemeRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	req.Normalize()
	if err := req.ValidatePartial(ctl.Validator); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	var m model.ClassEventThemeModel
	if err := ctl.DB.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("class_event_theme_id = ? AND class_event_theme_school_id = ? AND class_event_theme_deleted_at IS NULL", id, schoolID).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "resource not found")
		}
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	req.ApplyPatch(&m)

	if err := ctl.DB.Save(&m).Error; err != nil {
		if isDuplicateKey(err) {
			return helper.JsonError(c, http.StatusConflict, "theme code already exists for this school")
		}
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	return helper.JsonUpdated(c, "Updated", dto.FromModel(&m))
}

/*
=========================================================

	DELETE (soft)
	DELETE /api/a/:school_id/events/themes/:id

=========================================================
*/
func (ctl *ClassEventThemeController) Delete(c *fiber.Ctx) error {
	schoolID, err := ctl.resolveSchoolAndEnsureDKM(c)
	if err != nil {
		return nil
	}

	id, err := ctl.getID(c)
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	var m model.ClassEventThemeModel
	if err := ctl.DB.
		Where("class_event_theme_id = ? AND class_event_theme_school_id = ? AND class_event_theme_deleted_at IS NULL", id, schoolID).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "resource not found")
		}
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	if err := ctl.DB.Delete(&m).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	return helper.JsonDeleted(c, "Deleted", fiber.Map{
		"class_event_theme_id": id,
	})
}

/*
=========================================================

	UPSERT (by school_id, code)
	POST /api/a/:school_id/events/themes:upsert
	Body: CreateClassEventThemeRequest

=========================================================
*/
func (ctl *ClassEventThemeController) Upsert(c *fiber.Ctx) error {
	schoolID, err := ctl.resolveSchoolAndEnsureDKM(c)
	if err != nil {
		return nil
	}

	var req dto.CreateClassEventThemeRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	req.Normalize()
	if err := req.Validate(ctl.Validator); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	now := time.Now()

	m := req.ToModel(schoolID)
	m.ClassEventThemeUpdatedAt = now

	if err := ctl.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "class_event_theme_school_id"},
			{Name: "class_event_theme_code"},
		},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"class_event_theme_name":         m.ClassEventThemeName,
			"class_event_theme_color":        m.ClassEventThemeColor,
			"class_event_theme_custom_color": m.ClassEventThemeCustomColor,
			"class_event_theme_is_active":    m.ClassEventThemeIsActive,
			"class_event_theme_updated_at":   now,
		}),
	}).Create(m).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	var out model.ClassEventThemeModel
	if err := ctl.DB.
		Where("class_event_theme_school_id = ? AND class_event_theme_code = ? AND class_event_theme_deleted_at IS NULL",
			schoolID, m.ClassEventThemeCode).
		First(&out).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	return helper.JsonOK(c, "Upserted", dto.FromModel(&out))
}
