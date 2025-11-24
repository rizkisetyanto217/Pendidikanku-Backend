// internals/features/lembaga/ui/theme/controller/ui_theme_custom_preset_controller.go
package controller

import (
	"errors"
	"strings"
	"time"

	"madinahsalam_backend/internals/features/lembaga/ui/theme/dto"
	"madinahsalam_backend/internals/features/lembaga/ui/theme/model"
	helper "madinahsalam_backend/internals/helpers"

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
// internals/features/lembaga/ui/theme/controller/ui_theme_custom_preset_controller.go

/* =========================
   GET /ui-theme-custom-presets
   - Public list or get-by-id (?id=UUID)
   - Optional filter: ?school_id=UUID & ?q=keyword
   - Pagination & sort (whitelist):
       ?page=&per_page=&sort_by=created_at|updated_at|code|name&sort=asc|desc
========================= */

func (ctl *UIThemeCustomPresetController) Get(c *fiber.Ctx) error {
	// === Get by ID (single) ===
	if idStr := strings.TrimSpace(c.Query("id")); idStr != "" {
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
	// Pagination + sorting (konsisten dengan controller lain)
	p := helper.ParseFiber(c, "created_at", "desc", helper.DefaultOpts)

	// Whitelist kolom sort → nama kolom DB
	allowedSort := map[string]string{
		"created_at": "ui_theme_custom_preset_created_at",
		"updated_at": "ui_theme_custom_preset_updated_at",
		"code":       "ui_theme_custom_preset_code",
		"name":       "ui_theme_custom_preset_name",
	}
	orderClause, err := p.SafeOrderClause(allowedSort, "created_at")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid sort_by")
	}
	orderClause = strings.TrimPrefix(orderClause, "ORDER BY ")

	// Filters
	qkw := strings.TrimSpace(c.Query("q"))

	var schoolFilter uuid.UUID
	if s := strings.TrimSpace(c.Query("school_id")); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			schoolFilter = id
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "invalid school_id")
		}
	}

	// Build query
	dbq := ctl.DB.Model(&model.UIThemeCustomPreset{})
	if schoolFilter != uuid.Nil {
		dbq = dbq.Where("ui_theme_custom_preset_school_id = ?", schoolFilter)
	}
	if qkw != "" {
		like := "%" + qkw + "%"
		dbq = dbq.Where("(ui_theme_custom_preset_code ILIKE ? OR ui_theme_custom_preset_name ILIKE ?)", like, like)
	}

	// Count
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Fetch
	var rows []model.UIThemeCustomPreset
	if err := dbq.Order(orderClause).
		Offset(p.Offset()).
		Limit(p.Limit()).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Map ke response
	items := make([]dto.UIThemeCustomPresetResponse, 0, len(rows))
	for i := range rows {
		items = append(items, dto.ToUIThemeCustomPresetResponse(&rows[i]))
	}

	// 🔹 Pagination final via helper (JsonList auto-isi count & per_page_options)
	pg := helper.BuildPaginationFromOffset(total, p.Offset(), p.Limit())
	return helper.JsonList(c, "ok", items, pg)
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
