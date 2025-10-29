package controller

import (
	"fmt"
	dto "masjidku_backend/internals/features/finance/general_billings/dto"
	model "masjidku_backend/internals/features/finance/general_billings/model"
	helper "masjidku_backend/internals/helpers"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GET /api/a/:masjid_id/general-billings
// Query:
//
//	q, kind_id, active(=true|false|1|0), due_from(YYYY-MM-DD), due_to(YYYY-MM-DD),
//	include_global(=true|false)  -> default true
//	page, per_page, sort_by(created_at|due_date|title), order(asc|desc)
func (ctl *GeneralBillingController) List(c *fiber.Ctx) error {
	// === Path param ===
	midStr := strings.TrimSpace(c.Params("masjid_id"))
	if midStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "masjid_id wajib di path"})
	}
	mid, err := uuid.Parse(midStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "masjid_id tidak valid"})
	}

	// === Pagination & sorting ===
	p := helper.ParseFiber(c, "created_at", "desc", helper.DefaultOpts)
	allowed := map[string]string{
		"created_at": "general_billing_created_at",
		"due_date":   "general_billing_due_date",
		"title":      "general_billing_title",
	}
	col, ok := allowed[p.SortBy]
	if !ok {
		col = allowed["created_at"]
	}
	dir := "DESC"
	if strings.EqualFold(p.SortOrder, "asc") {
		dir = "ASC"
	}

	// === Filters ===
	q := strings.TrimSpace(c.Query("q"))
	kindID := strings.TrimSpace(c.Query("kind_id"))
	active := strings.TrimSpace(c.Query("active"))
	dueFrom := strings.TrimSpace(c.Query("due_from")) // YYYY-MM-DD
	dueTo := strings.TrimSpace(c.Query("due_to"))     // YYYY-MM-DD

	includeGlobal := true
	if v := strings.TrimSpace(c.Query("include_global")); v != "" {
		switch strings.ToLower(v) {
		case "false", "0", "no":
			includeGlobal = false
		}
	}

	// === Base query: alive + tenant/global scope ===
	db := ctl.DB.Model(&model.GeneralBilling{}).Where("general_billing_deleted_at IS NULL")

	if includeGlobal {
		// tampilkan PUNYA TENANT + GLOBAL
		db = db.Where("(general_billing_masjid_id = ? OR general_billing_masjid_id IS NULL)", mid)
	} else {
		// khusus PUNYA TENANT saja
		db = db.Where("general_billing_masjid_id = ?", mid)
	}

	// kind filter
	if kindID != "" {
		if uid, e := uuid.Parse(kindID); e == nil {
			db = db.Where("general_billing_kind_id = ?", uid)
		}
	}

	// active filter
	if active != "" {
		switch strings.ToLower(active) {
		case "true", "1", "yes":
			db = db.Where("general_billing_is_active = TRUE")
		case "false", "0", "no":
			db = db.Where("general_billing_is_active = FALSE")
		}
	}

	// due date range
	if dueFrom != "" {
		if t, e := time.Parse("2006-01-02", dueFrom); e == nil {
			db = db.Where("general_billing_due_date >= ?", t)
		}
	}
	if dueTo != "" {
		if t, e := time.Parse("2006-01-02", dueTo); e == nil {
			db = db.Where("general_billing_due_date <= ?", t)
		}
	}

	// free text search: code/title/desc
	if q != "" {
		pat := "%" + strings.ToLower(q) + "%"
		db = db.Where(`
			LOWER(COALESCE(general_billing_code, '')) LIKE ? OR
			LOWER(general_billing_title) LIKE ? OR
			LOWER(COALESCE(general_billing_desc, '')) LIKE ?
		`, pat, pat, pat)
	}

	// === Count ===
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// === Fetch (respect per_page=all) ===
	listQ := db.Order(fmt.Sprintf("%s %s", col, dir))
	if !p.All {
		listQ = listQ.Limit(p.Limit()).Offset(p.Offset())
	}

	var items []model.GeneralBilling
	if err := listQ.Find(&items).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	out := make([]*dto.GeneralBillingResponse, 0, len(items))
	for i := range items {
		out = append(out, dto.FromModelGeneralBilling(&items[i]))
	}

	return helper.JsonList(c, out, helper.BuildMeta(total, p))
}
