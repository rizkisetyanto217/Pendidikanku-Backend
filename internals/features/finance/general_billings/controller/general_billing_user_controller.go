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

// ========== List (paging + filter) ==========
// ListPublic: endpoint publik (opsional JWT). Tidak pakai EnsureMasjidAccessDKM.
func (ctl *GeneralBillingController) List(c *fiber.Ctx) error {
	// Ambil masjid_id dari path
	midStr := strings.TrimSpace(c.Params("masjid_id"))
	if midStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "masjid_id wajib di path"})
	}
	mid, err := uuid.Parse(midStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "masjid_id tidak valid"})
	}

	// === Pagination & sorting (helper) ===
	p := helper.ParseFiber(c, "created_at", "desc", helper.DefaultOpts)

	// whitelist kolom sorting → kolom DB sebenernya
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
	classID := strings.TrimSpace(c.Query("class_id"))
	sectionID := strings.TrimSpace(c.Query("section_id"))
	termID := strings.TrimSpace(c.Query("term_id"))
	active := strings.TrimSpace(c.Query("active"))
	dueFrom := strings.TrimSpace(c.Query("due_from")) // YYYY-MM-DD
	dueTo := strings.TrimSpace(c.Query("due_to"))     // YYYY-MM-DD

	db := model.ScopeAlive(ctl.DB).Scopes(model.ScopeByTenant(mid))

	if kindID != "" {
		if uid, e := uuid.Parse(kindID); e == nil {
			db = db.Scopes(model.ScopeByKind(uid))
		}
	}
	if classID != "" {
		if uid, e := uuid.Parse(classID); e == nil {
			db = db.Where("general_billing_class_id = ?", uid)
		}
	}
	if sectionID != "" {
		if uid, e := uuid.Parse(sectionID); e == nil {
			db = db.Where("general_billing_section_id = ?", uid)
		}
	}
	if termID != "" {
		if uid, e := uuid.Parse(termID); e == nil {
			db = db.Where("general_billing_term_id = ?", uid)
		}
	}
	if active != "" {
		switch active {
		case "true", "1":
			db = db.Where("general_billing_is_active = TRUE")
		case "false", "0":
			db = db.Where("general_billing_is_active = FALSE")
		}
	}
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
	if q != "" {
		pat := "%" + strings.ToLower(q) + "%"
		db = db.Where(`
			LOWER(COALESCE(general_billing_code, '')) LIKE ? OR
			LOWER(general_billing_title) LIKE ? OR
			LOWER(COALESCE(general_billing_desc, '')) LIKE ?
		`, pat, pat, pat)
	}

	// === Count + query ===
	var total int64
	if err := db.Model(&model.GeneralBilling{}).Count(&total).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	var items []model.GeneralBilling
	if err := db.
		Order(fmt.Sprintf("%s %s", col, dir)).
		Limit(p.Limit()).
		Offset(p.Offset()).
		Find(&items).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	out := make([]*dto.GeneralBillingResponse, 0, len(items))
	for i := range items {
		out = append(out, dto.FromModelGeneralBilling(&items[i]))
	}

	return c.JSON(fiber.Map{
		"data": out,
		"meta": helper.BuildMeta(total, p),
	})
}
