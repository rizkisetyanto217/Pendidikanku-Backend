// internals/features/lembaga/ui/theme/controller/ui_theme_choice_controller.go
package controller

import (
	"errors"
	"strings"
	"time"

	"schoolku_backend/internals/features/lembaga/ui/theme/dto"
	"schoolku_backend/internals/features/lembaga/ui/theme/model"
	helper "schoolku_backend/internals/helpers"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UIThemeChoiceController struct {
	DB       *gorm.DB
	Validate *validator.Validate
}

func NewUIThemeChoiceController(db *gorm.DB, validate *validator.Validate) *UIThemeChoiceController {
	return &UIThemeChoiceController{DB: db, Validate: validate}
}

/* =========================
   Helpers
========================= */

func boolQuery(c *fiber.Ctx, key string) *bool {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return nil
	}
	b := strings.EqualFold(raw, "true") || raw == "1"
	return &b
}

/* =========================
   POST /ui-theme-choices
   - Create new choice (exactly-one)
   - Jika is_default=true ⇒ nonaktifkan default lain di school tersebut (dalam TX)
========================= */

func (ctl *UIThemeChoiceController) Create(c *fiber.Ctx) error {
	var req dto.UIThemeChoiceRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid body")
	}
	if ctl.Validate != nil {
		if err := ctl.Validate.Struct(&req); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
		}
	}
	if err := req.ValidateCreate(); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	entity := model.UIThemeChoice{
		UIThemeChoiceSchoolID:       *req.UIThemeChoiceSchoolID,
		UIThemeChoicePresetID:       req.UIThemeChoicePresetID,
		UIThemeChoiceCustomPresetID: req.UIThemeChoiceCustomPresetID,
	}
	if req.UIThemeChoiceIsEnabled != nil {
		entity.UIThemeChoiceIsEnabled = *req.UIThemeChoiceIsEnabled
	} else {
		entity.UIThemeChoiceIsEnabled = true
	}
	if req.UIThemeChoiceIsDefault != nil {
		entity.UIThemeChoiceIsDefault = *req.UIThemeChoiceIsDefault
	}

	// TX: atur default unik per school bila diminta
	if err := ctl.DB.Transaction(func(tx *gorm.DB) error {
		if entity.UIThemeChoiceIsDefault {
			if err := tx.Model(&model.UIThemeChoice{}).
				Where("ui_theme_choice_school_id = ? AND ui_theme_choice_is_default = TRUE", entity.UIThemeChoiceSchoolID).
				Updates(map[string]interface{}{
					"ui_theme_choice_is_default": false,
					"ui_theme_choice_updated_at": time.Now(),
				}).Error; err != nil {
				return err
			}
		}
		return tx.Create(&entity).Error
	}); err != nil {
		if dto.IsUniqueViolation(err) {
			// Bisa bentrok karena unique partial index (default) / duplikat pasangan school-preset/custom
			return helper.JsonError(c, fiber.StatusConflict, "duplicate theme choice or default already set for this school")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonCreated(c, "theme choice created", dto.ToUIThemeChoiceResponse(&entity))
}

/* =========================
   GET /ui-theme-choices
   - ?id=UUID (single)
   - list + filter: ?school_id=UUID&preset_id=UUID&custom_preset_id=UUID&is_default=true|false&is_enabled=true|false
   - pagination: ?limit=&offset=
========================= */

func (ctl *UIThemeChoiceController) Get(c *fiber.Ctx) error {
	// Single by ID
	if idStr := c.Query("id"); idStr != "" {
		id, err := parseUUID(idStr)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "invalid id")
		}
		var entity model.UIThemeChoice
		if err := ctl.DB.First(&entity, "ui_theme_choice_id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusNotFound, "not found")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
		}
		return helper.JsonOK(c, "success get theme choice", dto.ToUIThemeChoiceResponse(&entity))
	}

	// List with filters
	limit := c.QueryInt("limit", 20)
	offset := c.QueryInt("offset", 0)

	// Filters
	var (
		schoolID, presetID, customID *uuid.UUID
	)
	if s := strings.TrimSpace(c.Query("school_id")); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			schoolID = &id
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "invalid school_id")
		}
	}
	if s := strings.TrimSpace(c.Query("preset_id")); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			presetID = &id
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "invalid preset_id")
		}
	}
	if s := strings.TrimSpace(c.Query("custom_preset_id")); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			customID = &id
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "invalid custom_preset_id")
		}
	}
	isDefault := boolQuery(c, "is_default")
	isEnabled := boolQuery(c, "is_enabled")

	dbq := ctl.DB.Model(&model.UIThemeChoice{})
	if schoolID != nil {
		dbq = dbq.Where("ui_theme_choice_school_id = ?", *schoolID)
	}
	if presetID != nil {
		dbq = dbq.Where("ui_theme_choice_preset_id = ?", *presetID)
	}
	if customID != nil {
		dbq = dbq.Where("ui_theme_choice_custom_preset_id = ?", *customID)
	}
	if isDefault != nil {
		dbq = dbq.Where("ui_theme_choice_is_default = ?", *isDefault)
	}
	if isEnabled != nil {
		dbq = dbq.Where("ui_theme_choice_is_enabled = ?", *isEnabled)
	}

	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	var rows []model.UIThemeChoice
	if err := dbq.Order("ui_theme_choice_created_at DESC").
		Limit(limit).Offset(offset).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	out := make([]dto.UIThemeChoiceResponse, 0, len(rows))
	for i := range rows {
		out = append(out, dto.ToUIThemeChoiceResponse(&rows[i]))
	}

	pagination := fiber.Map{
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}
	return helper.JsonList(c, out, pagination)
}

/* =========================
   PATCH /ui-theme-choices/:id
   - Partial update + aturan switch preset/custom
   - Jika is_default=true ⇒ nonaktifkan default lain di school tsb (TX)
========================= */

func (ctl *UIThemeChoiceController) Patch(c *fiber.Ctx) error {
	id, err := parseUUID(c.Params("id"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid id")
	}

	var req dto.UIThemeChoiceRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid body")
	}
	if ctl.Validate != nil {
		if err := ctl.Validate.Struct(&req); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
		}
	}
	if req.IsNoop() {
		return helper.JsonError(c, fiber.StatusBadRequest, "no fields to patch")
	}

	var entity model.UIThemeChoice
	if err := ctl.DB.First(&entity, "ui_theme_choice_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Terapkan patch ke copy supaya bisa tahu school_id final
	beforeSchool := entity.UIThemeChoiceSchoolID

	if err := dto.ApplyPatchToChoiceModel(&entity, &req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// TX: jika is_default true ⇒ reset default lainnya di school final
	if err := ctl.DB.Transaction(func(tx *gorm.DB) error {
		if entity.UIThemeChoiceIsDefault {
			if err := tx.Model(&model.UIThemeChoice{}).
				Where("ui_theme_choice_school_id = ? AND ui_theme_choice_id <> ? AND ui_theme_choice_is_default = TRUE",
					entity.UIThemeChoiceSchoolID, entity.UIThemeChoiceID).
				Updates(map[string]interface{}{
					"ui_theme_choice_is_default": false,
					"ui_theme_choice_updated_at": time.Now(),
				}).Error; err != nil {
				return err
			}
		}
		// Jika pindah school, amankan konsistensi (opsional)
		_ = beforeSchool
		return tx.Save(&entity).Error
	}); err != nil {
		if dto.IsUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "duplicate theme choice or default already set for this school")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonUpdated(c, "theme choice updated", dto.ToUIThemeChoiceResponse(&entity))
}

/* =========================
   DELETE /ui-theme-choices/:id
   - Hard delete
========================= */

func (ctl *UIThemeChoiceController) Delete(c *fiber.Ctx) error {
	id, err := parseUUID(c.Params("id"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid id")
	}

	res := ctl.DB.Delete(&model.UIThemeChoice{}, "ui_theme_choice_id = ?", id)
	if res.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, res.Error.Error())
	}
	if res.RowsAffected == 0 {
		return helper.JsonError(c, fiber.StatusNotFound, "not found")
	}

	return helper.JsonDeleted(c, "theme choice deleted", fiber.Map{
		"ui_theme_choice_id": id,
	})
}
