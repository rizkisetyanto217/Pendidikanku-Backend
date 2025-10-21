package controller

import (
	dto "masjidku_backend/internals/features/finance/general_billings/dto"
	m "masjidku_backend/internals/features/finance/general_billings/model"
	helperAuth "masjidku_backend/internals/helpers/auth"
	"strings"

	helper "masjidku_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
)

// List
// GET /api/a/:masjid_id/general-billing-kinds
func (ctl *GeneralBillingKindController) List(c *fiber.Ctx) error {
	// 1) Path + guard
	masjidID, err := helperAuth.ParseMasjidIDFromPath(c)
	if err != nil {
		return err
	}
	if er := helperAuth.EnsureDKMOrTeacherMasjid(c, masjidID); er != nil {
		return er
	}
	c.Locals("__masjid_guard_ok", masjidID.String())

	// 2) Query params
	var q dto.ListGeneralBillingKindsQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid query params")
	}
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}
	if q.PageSize > 100 {
		q.PageSize = 100
	}
	q.Search = strings.TrimSpace(q.Search)

	// 3) Build base query (alive only)
	tx := ctl.DB.WithContext(c.Context()).
		Model(&m.GeneralBillingKind{}).
		Where("general_billing_kind_masjid_id = ? AND general_billing_kind_deleted_at IS NULL", masjidID)

	// Filter: is_active
	if q.IsActive != nil {
		tx = tx.Where("general_billing_kind_is_active = ?", *q.IsActive)
	}

	// Filter: created_from / created_to
	if q.CreatedFrom != nil {
		tx = tx.Where("general_billing_kind_created_at >= ?", *q.CreatedFrom)
	}
	if q.CreatedTo != nil {
		tx = tx.Where("general_billing_kind_created_at < ?", *q.CreatedTo)
	}

	// Filter: search (code / name)
	if q.Search != "" {
		needle := "%" + strings.ToLower(q.Search) + "%"
		tx = tx.Where(
			"(LOWER(general_billing_kind_code) LIKE ? OR LOWER(general_billing_kind_name) LIKE ?)",
			needle, needle,
		)
	}

	// 4) Count total
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// 5) Sorting whitelist
	order := "general_billing_kind_created_at DESC" // default
	switch strings.ToLower(strings.TrimSpace(q.Sort)) {
	case "created_at_asc":
		order = "general_billing_kind_created_at ASC"
	case "created_at_desc":
		order = "general_billing_kind_created_at DESC"
	case "name_asc":
		order = "general_billing_kind_name ASC"
	case "name_desc":
		order = "general_billing_kind_name DESC"
	}

	// 6) Paging
	offset := (q.Page - 1) * q.PageSize

	// 7) Fetch data
	var rows []m.GeneralBillingKind
	if err := tx.
		Order(order).
		Offset(offset).
		Limit(q.PageSize).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// 8) Build pagination meta
	totalPages := (total + int64(q.PageSize) - 1) / int64(q.PageSize)
	pagination := fiber.Map{
		"page":        q.Page,
		"page_size":   q.PageSize,
		"total":       total,
		"total_pages": totalPages,
	}

	// 9) Response
	return helper.JsonList(c, dto.FromModelSlice(rows), pagination)
}
