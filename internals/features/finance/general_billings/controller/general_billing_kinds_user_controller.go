package controller

import (
	"strings"
	"time"

	dto "schoolku_backend/internals/features/finance/general_billings/dto"
	m "schoolku_backend/internals/features/finance/general_billings/model"
	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
)

// GET /api/a/:school_id/general-billing-kinds
func (ctl *GeneralBillingKindController) List(c *fiber.Ctx) error {
	// 1) Guard school di path
	schoolID, err := helperAuth.ParseSchoolIDFromPath(c)
	if err != nil {
		return err
	}
	if er := helperAuth.EnsureDKMOrTeacherSchool(c, schoolID); er != nil {
		return er
	}
	c.Locals("__school_guard_ok", schoolID.String())

	// 2) Ambil query (non-paging)
	var q dto.ListGeneralBillingKindsQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid query params")
	}
	q.Search = strings.TrimSpace(q.Search)

	// 2a) Fallback tanggal (dukungan "YYYY-MM-DD" selain RFC3339)
	if q.CreatedFrom == nil {
		if t := parseTimePtrLoose(c.Query("created_from")); t != nil {
			q.CreatedFrom = t
		}
	}
	if q.CreatedTo == nil {
		if t := parseTimePtrLoose(c.Query("created_to")); t != nil {
			q.CreatedTo = t
		}
	}

	// 3) Paging + sorting (default sort_by=created_at, order=desc)
	p := helper.ParseFiber(c, "created_at", "desc", helper.DefaultOpts)

	// 3a) whitelist sort → kolom DB
	allowed := map[string]string{
		"created_at": "general_billing_kind_created_at",
		"name":       "general_billing_kind_name",
		"code":       "general_billing_kind_code",
	}

	// 3b) ambil clause dari helper, lalu buang "ORDER BY " supaya cocok untuk GORM.Order()
	orderClause, err := p.SafeOrderClause(allowed, "created_at")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	orderExpr := sanitizeOrderForGorm(orderClause) // ← "col DESC" tanpa "ORDER BY "
	if orderExpr == "" {
		orderExpr = "general_billing_kind_created_at DESC"
	}

	// 4) Base query: tenant + belum dihapus
	tx := ctl.DB.WithContext(c.Context()).
		Model(&m.GeneralBillingKind{}).
		Where("general_billing_kind_school_id = ? AND general_billing_kind_deleted_at IS NULL", schoolID)

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
	if q.Category != nil && strings.TrimSpace(*q.Category) != "" {
		tx = tx.Where("general_billing_kind_category = ?", strings.TrimSpace(*q.Category))
	}
	if q.IsGlobal != nil {
		tx = tx.Where("general_billing_kind_is_global = ?", *q.IsGlobal)
	}
	if q.Visible != nil && strings.TrimSpace(*q.Visible) != "" {
		tx = tx.Where("general_billing_kind_visibility = ?", strings.TrimSpace(*q.Visible))
	}
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
		tx = tx.Where("(LOWER(general_billing_kind_code) LIKE ? OR LOWER(general_billing_kind_name) LIKE ?)", needle, needle)
	}

	// 6) Hitung total
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// 7) Ambil data + order + paging
	var rows []m.GeneralBillingKind
	if err := tx.
		Order(orderExpr).                      // aman: tidak ada "ORDER BY " ganda
		Order("general_billing_kind_id DESC"). // tie-breaker biar paging stabil
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

// -------- helpers lokal --------

// sanitizeOrderForGorm membuang prefix "ORDER BY " (case-insensitive) dari output SafeOrderClause
// sehingga bisa langsung dipakai di GORM.Order()
func sanitizeOrderForGorm(clause string) string {
	s := strings.TrimSpace(clause)
	if s == "" {
		return ""
	}
	up := strings.ToUpper(s)
	const prefix = "ORDER BY "
	if strings.HasPrefix(up, prefix) {
		// potong sepanjang prefix sesuai case asli
		return strings.TrimSpace(s[len(prefix):])
	}
	return s
}

// parseTimePtrLoose: dukung "YYYY-MM-DD" dan RFC3339
func parseTimePtrLoose(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	// YYYY-MM-DD (anggap awal hari local → aman untuk perbandingan >=)
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return &t
	}
	// RFC3339
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return &t
	}
	return nil
}
