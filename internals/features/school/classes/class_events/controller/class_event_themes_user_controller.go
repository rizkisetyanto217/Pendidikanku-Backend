// file: internals/features/school/classes/class_events/controller/class_event_theme_controller.go
package controller

import (
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"

	dto "madinahsalam_backend/internals/features/school/classes/class_events/dto"
	model "madinahsalam_backend/internals/features/school/classes/class_events/model"
	helper "madinahsalam_backend/internals/helpers"
)

/*
=========================================================

	LIST
	GET /api/a/:school_id/events/themes
	Query:
	- q
	- is_active (true|false)
	- page, per_page  (atau limit/offset juga didukung oleh helper)
	- sort_by (created_at|updated_at|name|is_active)
	- order (asc|desc)
	- (kompat) order_by → dipetakan ke sort_by

=========================================================
*/
func (ctl *ClassEventThemeController) List(c *fiber.Ctx) error {
	schoolID, err := ctl.resolveSchoolAndEnsureDKM(c)
	if err != nil {
		return err // <- kembalikan error (sebelumnya 'return nil')
	}

	// --- parse pagination + sorting (pakai helper) ---
	// default: created_at DESC
	p := helper.ParseFiber(c, "created_at", "desc", helper.AdminOpts)

	// kompat lama: "order_by" → sort_by
	if ob := strings.TrimSpace(c.Query("order_by")); ob != "" {
		p.SortBy = ob
	}

	// whitelist kolom untuk ORDER BY
	allowed := map[string]string{
		"created_at": "class_event_theme_created_at",
		"updated_at": "class_event_theme_updated_at",
		"name":       "class_event_theme_name",
		"is_active":  "class_event_theme_is_active",
	}
	orderClause, err := p.SafeOrderClause(allowed, "created_at")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	// GORM Order() tidak butuh "ORDER BY "
	orderExpr := strings.TrimPrefix(orderClause, "ORDER BY ")

	// --- filters ringan ---
	q := strings.TrimSpace(c.Query("q"))
	isActiveStr := strings.ToLower(strings.TrimSpace(c.Query("is_active")))
	var isActive *bool
	if isActiveStr == "true" || isActiveStr == "false" {
		b := isActiveStr == "true"
		isActive = &b
	}

	// --- query ---
	tx := ctl.DB.
		Model(&model.ClassEventThemeModel{}).
		Where("class_event_theme_school_id = ? AND class_event_theme_deleted_at IS NULL", schoolID)

	if isActive != nil {
		tx = tx.Where("class_event_theme_is_active = ?", *isActive)
	}
	if q != "" {
		tx = tx.Where("class_event_theme_name ILIKE ?", "%"+q+"%")
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	var rows []model.ClassEventThemeModel
	if err := tx.
		Order(orderExpr).
		Limit(p.Limit()).
		Offset(p.Offset()).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	// 🔹 pagination seragam (auto has_next/has_prev, dll.)
	pg := helper.BuildPaginationFromOffset(total, p.Offset(), p.Limit())

	// 🔹 response seragam (JsonList isi "message", "data", "pagination")
	return helper.JsonList(c, "ok", dto.FromModels(rows), pg)
}