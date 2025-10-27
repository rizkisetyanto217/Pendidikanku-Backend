package controller

import (
	dto "masjidku_backend/internals/features/finance/general_billings/dto"
	m "masjidku_backend/internals/features/finance/general_billings/model"
	helperAuth "masjidku_backend/internals/helpers/auth"
	"strings"

	helper "masjidku_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
)

// GET /api/a/:masjid_id/general-billing-kinds
func (ctl *GeneralBillingKindController) List(c *fiber.Ctx) error {
	// 1) Path guard
	masjidID, err := helperAuth.ParseMasjidIDFromPath(c)
	if err != nil {
		return err
	}
	if er := helperAuth.EnsureDKMOrTeacherMasjid(c, masjidID); er != nil {
		return er
	}
	c.Locals("__masjid_guard_ok", masjidID.String())

	// 2) Query params (filter2 non-paging)
	var q dto.ListGeneralBillingKindsQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid query params")
	}
	q.Search = strings.TrimSpace(q.Search)

	// 3) Pagination & sorting (pakai helper)
	// - default sort_by = "created_at", default order = "desc"
	// - ganti ExportOpts jika ingin dukung per_page=all
	p := helper.ParseFiber(c, "created_at", "desc", helper.DefaultOpts)

	// whitelist mapping: key -> kolom DB
	allowed := map[string]string{
		"created_at": "general_billing_kind_created_at",
		"name":       "general_billing_kind_name",
		"code":       "general_billing_kind_code",
	}
	orderClause, err := p.SafeOrderClause(allowed, "created_at")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// 4) Base query (alive + tenant)
	tx := ctl.DB.WithContext(c.Context()).
		Model(&m.GeneralBillingKind{}).
		Where("general_billing_kind_masjid_id = ? AND general_billing_kind_deleted_at IS NULL", masjidID)

	// 5) Filters tambahan
	if q.IsActive != nil {
		tx = tx.Where("general_billing_kind_is_active = ?", *q.IsActive)
	}
	if q.CreatedFrom != nil {
		tx = tx.Where("general_billing_kind_created_at >= ?", *q.CreatedFrom)
	}
	if q.CreatedTo != nil {
		tx = tx.Where("general_billing_kind_created_at < ?", *q.CreatedTo)
	}
	if q.Category != nil && *q.Category != "" {
		tx = tx.Where("general_billing_kind_category = ?", *q.Category)
	}
	if q.IsGlobal != nil {
		tx = tx.Where("general_billing_kind_is_global = ?", *q.IsGlobal)
	}
	if q.Visible != nil && *q.Visible != "" {
		tx = tx.Where("general_billing_kind_visibility = ?", *q.Visible)
	}
	// Flags (baru)
	if q.IsRecurring != nil {
		tx = tx.Where("general_billing_kind_is_recurring = ?", *q.IsRecurring)
	}
	if q.RequiresMonthYear != nil {
		tx = tx.Where("general_billing_kind_requires_month_year = ?", *q.RequiresMonthYear)
	}
	if q.RequiresOptionCode != nil {
		tx = tx.Where("general_billing_kind_requires_option_code = ?", *q.RequiresOptionCode)
	}

	// Search (code/name)
	if q.Search != "" {
		needle := "%" + strings.ToLower(q.Search) + "%"
		tx = tx.Where(
			"(LOWER(general_billing_kind_code) LIKE ? OR LOWER(general_billing_kind_name) LIKE ?)",
			needle, needle,
		)
	}

	// 6) Hitung total
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// 7) Ambil data dengan order & paging dari helper
	var rows []m.GeneralBillingKind
	if err := tx.
		Order(orderClause).
		Offset(p.Offset()).
		Limit(p.Limit()).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// 8) Meta pagination
	meta := helper.BuildMeta(total, p)

	// 9) Response
	return helper.JsonList(c, dto.FromModelSlice(rows), meta)
}
