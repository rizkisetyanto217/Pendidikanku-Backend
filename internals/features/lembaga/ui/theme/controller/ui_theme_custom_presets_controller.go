// internals/features/lembaga/ui/theme/controller/ui_theme_custom_preset_controller.go
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

type UIThemeCustomPresetController struct {
	DB       *gorm.DB
	Validate *validator.Validate
}

func NewUIThemeCustomPresetController(db *gorm.DB, validate *validator.Validate) *UIThemeCustomPresetController {
	return &UIThemeCustomPresetController{DB: db, Validate: validate}
}

/* =========================
   Helpers
========================= */

func parseUUID(str string) (uuid.UUID, error) {
	if strings.TrimSpace(str) == "" {
		return uuid.Nil, errors.New("empty id")
	}
	return uuid.Parse(str)
}

// Detect Postgres unique violation (23505)
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key value violates unique constraint") ||
		strings.Contains(msg, "sqlstate 23505") ||
		strings.Contains(msg, "duplicate key")
}

/* =========================
   POST /ui-theme-custom-presets
   - Create (OWNER via middleware di routes)
========================= */

func (ctl *UIThemeCustomPresetController) Create(c *fiber.Ctx) error {
	var req dto.UIThemeCustomPresetRequest
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

	entity := model.UIThemeCustomPreset{
		UIThemeCustomPresetSchoolID: *req.UIThemeCustomPresetSchoolID,
		UIThemeCustomPresetCode:     *req.UIThemeCustomPresetCode,
		UIThemeCustomPresetName:     *req.UIThemeCustomPresetName,
		UIThemeCustomPresetLight:    *req.UIThemeCustomPresetLight,
		UIThemeCustomPresetDark:     *req.UIThemeCustomPresetDark,
	}
	// optional
	if req.UIThemeCustomBasePresetID != nil {
		entity.UIThemeCustomBasePresetID = req.UIThemeCustomBasePresetID
	}
	if req.UIThemeCustomPresetIsActive != nil {
		entity.UIThemeCustomPresetIsActive = *req.UIThemeCustomPresetIsActive
	}

	if err := ctl.DB.Create(&entity).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "custom preset code already exists for this school")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonCreated(c, "custom theme preset created", dto.ToUIThemeCustomPresetResponse(&entity))
}

/* =========================
   GET /ui-theme-custom-presets
   - Public list or get-by-id (?id=UUID)
   - Optional filter: ?school_id=UUID & ?q=keyword
========================= */

func (ctl *UIThemeCustomPresetController) Get(c *fiber.Ctx) error {
	if idStr := c.Query("id"); idStr != "" {
		// === Get by ID ===
		id, err := parseUUID(idStr)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "invalid id")
		}
		var entity model.UIThemeCustomPreset
		if err := ctl.DB.First(&entity, "ui_theme_custom_preset_id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusNotFound, "not found")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
		}
		return helper.JsonOK(c, "success get custom theme preset", dto.ToUIThemeCustomPresetResponse(&entity))
	}

	// === List ===
	q := strings.TrimSpace(c.Query("q"))
	limit := c.QueryInt("limit", 20)
	offset := c.QueryInt("offset", 0)

	var schoolFilter uuid.UUID
	if s := strings.TrimSpace(c.Query("school_id")); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			schoolFilter = id
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "invalid school_id")
		}
	}

	var rows []model.UIThemeCustomPreset
	dbq := ctl.DB.Model(&model.UIThemeCustomPreset{})

	if schoolFilter != uuid.Nil {
		dbq = dbq.Where("ui_theme_custom_preset_school_id = ?", schoolFilter)
	}
	if q != "" {
		like := "%" + q + "%"
		dbq = dbq.Where("(ui_theme_custom_preset_code ILIKE ? OR ui_theme_custom_preset_name ILIKE ?)", like, like)
	}

	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	if err := dbq.Order("ui_theme_custom_preset_created_at DESC").
		Limit(limit).Offset(offset).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	out := make([]dto.UIThemeCustomPresetResponse, 0, len(rows))
	for i := range rows {
		out = append(out, dto.ToUIThemeCustomPresetResponse(&rows[i]))
	}

	pagination := fiber.Map{
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}
	return helper.JsonList(c, out, pagination)
}

/* =========================
   PATCH /ui-theme-custom-presets/:id
   - OWNER via middleware
========================= */

func (ctl *UIThemeCustomPresetController) Patch(c *fiber.Ctx) error {
	id, err := parseUUID(c.Params("id"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid id")
	}

	var req dto.UIThemeCustomPresetRequest
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

	var entity model.UIThemeCustomPreset
	if err := ctl.DB.First(&entity, "ui_theme_custom_preset_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Terapkan patch (scalar + JSON replace + JSON merge)
	if err := dto.ApplyPatchToCustomModel(&entity, &req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Pastikan updated_at terisi (guard kedua)
	entity.UIThemeCustomPresetUpdatedAt = time.Now()

	if err := ctl.DB.Save(&entity).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "custom preset code already exists for this school")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonUpdated(c, "custom theme preset updated", dto.ToUIThemeCustomPresetResponse(&entity))
}

/* =========================
   DELETE /ui-theme-custom-presets/:id
   - OWNER via middleware
   - Hard delete (karena kolom deleted_at tidak ada di schema)
========================= */

func (ctl *UIThemeCustomPresetController) Delete(c *fiber.Ctx) error {
	id, err := parseUUID(c.Params("id"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid id")
	}

	res := ctl.DB.Delete(&model.UIThemeCustomPreset{}, "ui_theme_custom_preset_id = ?", id)
	if res.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, res.Error.Error())
	}
	if res.RowsAffected == 0 {
		return helper.JsonError(c, fiber.StatusNotFound, "not found")
	}

	return helper.JsonDeleted(c, "custom theme preset deleted", fiber.Map{
		"ui_theme_custom_preset_id": id,
	})
}
