package controller

import (
	"errors"
	dto "masjidku_backend/internals/features/finance/general_billings/dto"
	model "masjidku_backend/internals/features/finance/general_billings/model"
	helperAuth "masjidku_backend/internals/helpers/auth"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ========== GetByID ==========
func (ctl *GeneralBillingController) GetByID(c *fiber.Ctx) error {
	if !(helperAuth.IsOwner(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Akses ditolak"})
	}

	idStr := strings.TrimSpace(c.Params("id"))
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "general_billing_id invalid"})
	}

	var gb model.GeneralBilling
	if err := model.ScopeAlive(ctl.DB).
		First(&gb, "general_billing_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Data tidak ditemukan"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Tenant guard
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Masjid context tidak valid"})
	}
	mid, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}
	if gb.GeneralBillingMasjidID != mid {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Tidak boleh mengakses data tenant lain"})
	}

	return c.JSON(dto.FromModelGeneralBilling(&gb))
}

// ========== List (paging + filter) ==========
func (ctl *GeneralBillingController) List(c *fiber.Ctx) error {
	if !(helperAuth.IsOwner(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Akses ditolak"})
	}

	// Tenant scope
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Masjid context tidak valid"})
	}
	mid, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}

	// Query params
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit <= 0 || limit > 200 {
		limit = 20
	}

	q := strings.TrimSpace(c.Query("q"))
	kindID := strings.TrimSpace(c.Query("kind_id"))
	classID := strings.TrimSpace(c.Query("class_id"))
	sectionID := strings.TrimSpace(c.Query("section_id"))
	termID := strings.TrimSpace(c.Query("term_id"))
	active := strings.TrimSpace(c.Query("active"))
	dueFrom := strings.TrimSpace(c.Query("due_from")) // "YYYY-MM-DD"
	dueTo := strings.TrimSpace(c.Query("due_to"))     // "YYYY-MM-DD"
	sort := strings.TrimSpace(c.Query("sort"))        // default: created_at desc

	db := model.ScopeAlive(ctl.DB).
		Scopes(model.ScopeByTenant(mid))

	// Filters
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
		if active == "true" || active == "1" {
			db = db.Where("general_billing_is_active = TRUE")
		} else if active == "false" || active == "0" {
			db = db.Where("general_billing_is_active = FALSE")
		}
	}
	// Due date range
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
	// Simple search (code/title/desc)
	if q != "" {
		pat := "%" + strings.ToLower(q) + "%"
		db = db.Where(`
			LOWER(COALESCE(general_billing_code, '')) LIKE ? OR
			LOWER(general_billing_title) LIKE ? OR
			LOWER(COALESCE(general_billing_desc, '')) LIKE ?
		`, pat, pat, pat)
	}

	// Sorting
	switch sort {
	case "due_date_asc":
		db = db.Order("general_billing_due_date ASC NULLS LAST")
	case "due_date_desc":
		db = db.Order("general_billing_due_date DESC NULLS LAST")
	case "title_asc":
		db = db.Order("general_billing_title ASC")
	case "title_desc":
		db = db.Order("general_billing_title DESC")
	case "created_asc":
		db = db.Order("general_billing_created_at ASC")
	default:
		db = db.Order("general_billing_created_at DESC")
	}

	// Pagination
	var total int64
	if err := db.Model(&model.GeneralBilling{}).Count(&total).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	var items []model.GeneralBilling
	if err := db.
		Limit(limit).
		Offset((page - 1) * limit).
		Find(&items).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	out := make([]*dto.GeneralBillingResponse, 0, len(items))
	for i := range items {
		out = append(out, dto.FromModelGeneralBilling(&items[i]))
	}

	return c.JSON(fiber.Map{
		"data":       out,
		"page":       page,
		"limit":      limit,
		"total":      total,
		"total_page": (total + int64(limit) - 1) / int64(limit),
	})
}
