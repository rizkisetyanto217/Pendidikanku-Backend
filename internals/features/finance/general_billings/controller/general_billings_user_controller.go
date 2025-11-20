package controller

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	dto "schoolku_backend/internals/features/finance/general_billings/dto"
	model "schoolku_backend/internals/features/finance/general_billings/model"
	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GET /api/a/:school_id/general-billings
// Query:
//
//	q, kind_id, active(=true|false|1|0), due_from(YYYY-MM-DD), due_to(YYYY-MM-DD),
//	include_global(=true|false)  -> default true
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
		// legacy: dari path (boleh UUID / slug, tergantung ParseSchoolIDFromPath kamu)
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
		db = db.Where("(general_billing_school_id = ? OR general_billing_school_id IS NULL)", schoolID)
	} else {
		// khusus PUNYA TENANT saja
		db = db.Where("general_billing_school_id = ?", schoolID)
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
	if !allMode {
		listQ = listQ.Limit(pg.Limit).Offset(pg.Offset)
	}

	var items []model.GeneralBilling
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
