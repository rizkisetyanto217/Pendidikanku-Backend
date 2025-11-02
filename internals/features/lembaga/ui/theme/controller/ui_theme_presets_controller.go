// internals/features/lembaga/ui/theme/controller/ui_theme_preset_controller.go
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
	"gorm.io/gorm"
)

type UIThemePresetController struct {
	DB       *gorm.DB
	Validate *validator.Validate
}

func NewUIThemePresetController(db *gorm.DB, validate *validator.Validate) *UIThemePresetController {
	return &UIThemePresetController{DB: db, Validate: validate}
}

/* =========================
   POST /ui-theme-presets
   - Create new preset
========================= */

func (ctl *UIThemePresetController) Create(c *fiber.Ctx) error {
	var req dto.UIThemePresetRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid body")
	}
	// validator tags (opsional)
	if ctl.Validate != nil {
		if err := ctl.Validate.Struct(&req); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
		}
	}
	// pastikan field wajib untuk CREATE
	if err := req.ValidateCreate(); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	entity := model.UIThemePreset{
		UIThemePresetCode:  *req.UIThemePresetCode,
		UIThemePresetName:  *req.UIThemePresetName,
		UIThemePresetLight: *req.UIThemePresetLight,
		UIThemePresetDark:  *req.UIThemePresetDark,
	}

	if err := ctl.DB.Create(&entity).Error; err != nil {
		if dto.IsUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "ui_theme_preset_code already exists")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonCreated(c, "theme preset created", dto.ToUIThemePresetResponse(&entity))
}

/* =========================
   GET (list or single)
========================= */

func (ctl *UIThemePresetController) Get(c *fiber.Ctx) error {
	idStr := c.Query("id")
	if idStr != "" {
		// === Get by ID ===
		id, err := parseUUID(idStr)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "invalid id")
		}

		var entity model.UIThemePreset
		if err := ctl.DB.First(&entity, "ui_theme_preset_id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusNotFound, "not found")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
		}

		return helper.JsonOK(c, "success get theme preset", dto.ToUIThemePresetResponse(&entity))
	}

	// === List ===
	q := strings.TrimSpace(c.Query("q"))
	limit := c.QueryInt("limit", 20)
	offset := c.QueryInt("offset", 0)
	includeDeleted := strings.EqualFold(c.Query("include_deleted"), "true")

	var rows []model.UIThemePreset
	dbq := ctl.DB.Model(&model.UIThemePreset{})

	if !includeDeleted {
		dbq = dbq.Where("ui_theme_preset_deleted_at IS NULL")
	}
	if q != "" {
		like := "%" + q + "%"
		dbq = dbq.Where("(ui_theme_preset_code ILIKE ? OR ui_theme_preset_name ILIKE ?)", like, like)
	}

	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	if err := dbq.Order("ui_theme_preset_created_at DESC").
		Limit(limit).Offset(offset).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	out := make([]dto.UIThemePresetResponse, 0, len(rows))
	for i := range rows {
		out = append(out, dto.ToUIThemePresetResponse(&rows[i]))
	}

	pagination := fiber.Map{
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}
	return helper.JsonList(c, out, pagination)
}

/* =========================
   PATCH (partial + JSON merge)
========================= */

func (ctl *UIThemePresetController) Patch(c *fiber.Ctx) error {
	id, err := parseUUID(c.Params("id"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid id")
	}

	var req dto.UIThemePresetRequest
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

	var entity model.UIThemePreset
	if err := ctl.DB.First(&entity, "ui_theme_preset_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Terapkan patch (scalar + JSON replace + JSON merge RFC7386)
	if err := dto.ApplyPatchToModel(&entity, &req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	if err := ctl.DB.Save(&entity).Error; err != nil {
		if dto.IsUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "ui_theme_preset_code already exists")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonUpdated(c, "theme preset updated", dto.ToUIThemePresetResponse(&entity))
}

/* =========================
   DELETE (soft delete)
========================= */

func (ctl *UIThemePresetController) Delete(c *fiber.Ctx) error {
	id, err := parseUUID(c.Params("id"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid id")
	}

	// Soft delete: set deleted_at + updated_at
	now := time.Now()
	res := ctl.DB.Model(&model.UIThemePreset{}).
		Where("ui_theme_preset_id = ? AND ui_theme_preset_deleted_at IS NULL", id).
		Updates(map[string]interface{}{
			"ui_theme_preset_deleted_at": now,
			"ui_theme_preset_updated_at": now,
		})

	if res.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, res.Error.Error())
	}
	if res.RowsAffected == 0 {
		return helper.JsonError(c, fiber.StatusNotFound, "not found or already deleted")
	}

	// Opsional: kembalikan hanya id agar ringan
	return helper.JsonDeleted(c, "theme preset deleted", fiber.Map{
		"ui_theme_preset_id": id,
	})
}
