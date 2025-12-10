package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	dto "madinahsalam_backend/internals/features/finance/general_billings/dto"
	model "madinahsalam_backend/internals/features/finance/general_billings/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// helper kecil lokal, pengganti helper.AtoiSafe
func atoiSafe(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	return strconv.Atoi(s)
}

// GET /api/a/:school_id/general-billings
// Query:
//
//	q
//	active(=true|false|1|0)
//	due_from(YYYY-MM-DD)
//	due_to(YYYY-MM-DD)
//	category (registration|spp|mass_student|donation)
//	bill_code (SPP / lainnya)
//	month (1-12)
//	year (2000-2100)
//	page, per_page(atau limit), sort_by(created_at|due_date|title), order(asc|desc)
//	per_page=all  -> ambil semua (tanpa limit/offset)
func (ctl *GeneralBillingController) List(c *fiber.Ctx) error {
	// === Resolve school context: token > active-school > path ===
	var schoolID uuid.UUID
	if id, err := helperAuth.GetSchoolIDFromTokenPreferTeacher(c); err == nil && id != uuid.Nil {
		schoolID = id
	} else if id, err := helperAuth.GetActiveSchoolID(c); err == nil && id != uuid.Nil {
		schoolID = id
	} else {
		sid, err := helperAuth.ParseSchoolIDFromPath(c)
		if err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "school context not found")
		}
		schoolID = sid
	}

	// === Guard: hanya staff (teacher/dkm/admin/bendahara) ===
	if err := helperAuth.EnsureStaffSchool(c, schoolID); err != nil {
		return err
	}
	c.Locals("__school_guard_ok", schoolID.String())

	// === Pagination & sorting ===
	pg := helper.ResolvePaging(c, 20, 200)

	// dukung per_page=all (tanpa limit/offset)
	perPageRaw := strings.ToLower(strings.TrimSpace(c.Query("per_page")))
	allMode := perPageRaw == "all"

	allowed := map[string]string{
		"created_at": "general_billing_created_at",
		"due_date":   "general_billing_due_date",
		"title":      "general_billing_title",
	}
	sortBy := strings.ToLower(strings.TrimSpace(c.Query("sort_by")))
	col, ok := allowed[sortBy]
	if !ok {
		col = allowed["created_at"]
	}
	dir := "DESC"
	if strings.EqualFold(strings.TrimSpace(c.Query("order")), "asc") {
		dir = "ASC"
	}

	// === Filters ===
	q := strings.TrimSpace(c.Query("q"))
	active := strings.TrimSpace(c.Query("active"))
	dueFrom := strings.TrimSpace(c.Query("due_from")) // YYYY-MM-DD
	dueTo := strings.TrimSpace(c.Query("due_to"))     // YYYY-MM-DD

	categoryStr := strings.TrimSpace(c.Query("category")) // registration|spp|mass_student|donation
	billCode := strings.TrimSpace(c.Query("bill_code"))   // SPP / lain
	monthStr := strings.TrimSpace(c.Query("month"))       // 1-12
	yearStr := strings.TrimSpace(c.Query("year"))         // 2000-2100

	// === Base query: alive + tenant-scope ===
	db := ctl.DB.Model(&model.GeneralBillingModel{}).
		Where("general_billing_deleted_at IS NULL").
		Where("general_billing_school_id = ?", schoolID)

	// category (enum string)
	if categoryStr != "" {
		db = db.Where("general_billing_category = ?", categoryStr)
	}

	// bill_code
	if billCode != "" {
		db = db.Where("general_billing_bill_code = ?", billCode)
	}

	// month/year
	if monthStr != "" {
		if m, err := atoiSafe(monthStr); err == nil && m >= 1 && m <= 12 {
			db = db.Where("general_billing_month = ?", m)
		}
	}
	if yearStr != "" {
		if y, err := atoiSafe(yearStr); err == nil && y >= 2000 && y <= 2100 {
			db = db.Where("general_billing_year = ?", y)
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

	// free text search: code/title/desc/bill_code
	if q != "" {
		pat := "%" + strings.ToLower(q) + "%"
		db = db.Where(`
			LOWER(COALESCE(general_billing_code, '')) LIKE ? OR
			LOWER(general_billing_title) LIKE ? OR
			LOWER(COALESCE(general_billing_desc, '')) LIKE ? OR
			LOWER(COALESCE(general_billing_bill_code, '')) LIKE ?
		`, pat, pat, pat, pat)
	}

	// === Count ===
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// === Fetch (respect per_page=all) ===
	listQ := db.Order(fmt.Sprintf("%s %s", col, dir))
	if !allMode {
		listQ = listQ.Limit(pg.Limit).Offset(pg.Offset)
	}

	var items []model.GeneralBillingModel
	if err := listQ.Find(&items).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	out := make([]*dto.GeneralBillingResponse, 0, len(items))
	for i := range items {
		out = append(out, dto.FromModelGeneralBilling(&items[i]))
	}

	// === Build pagination untuk JsonList ===
	var pagination helper.Pagination
	if allMode {
		pagination = helper.BuildPaginationFromPage(total, 1, int(total))
	} else {
		pagination = helper.BuildPaginationFromPage(total, pg.Page, pg.PerPage)
	}

	return helper.JsonList(c, "OK", out, pagination)
}
