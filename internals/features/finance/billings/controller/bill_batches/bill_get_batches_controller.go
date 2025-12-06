package controller

import (
	"fmt"
	"net/http"
	"strings"

	"madinahsalam_backend/internals/features/finance/billings/dto"
	billing "madinahsalam_backend/internals/features/finance/billings/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// =======================================================
// LIST (filters + pagination, tenant-scoped by school)
// school_id diambil dari token/context
// GET /spp/bill-batches
// =======================================================

func (h *BillBatchHandler) ListBillBatches(c *fiber.Ctx) error {
	// === Resolve school context dari token/context ===
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		// ResolveSchoolIDFromContext sudah balikin JsonError yang rapi
		return err
	}

	// === Guard: hanya staff (teacher/dkm/admin/bendahara) ===
	if err := helperAuth.EnsureStaffSchool(c, schoolID); err != nil {
		return err
	}

	// === Paging (default 20, max 200) + dukungan per_page=all ===
	pg := helper.ResolvePaging(c, 20, 200)
	perPageRaw := strings.ToLower(strings.TrimSpace(c.Query("per_page")))
	allMode := perPageRaw == "all"
	offset := (pg.Page - 1) * pg.PerPage

	// Base query: tenant-scoped + belum dihapus
	q := h.DB.Model(&billing.BillBatch{}).
		Where("bill_batch_school_id = ? AND bill_batch_deleted_at IS NULL", schoolID)

	// === Filters tambahan ===

	// class_id
	if s := strings.TrimSpace(c.Query("class_id")); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			q = q.Where("bill_batch_class_id = ?", id)
		}
	}

	// section_id
	if s := strings.TrimSpace(c.Query("section_id")); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			q = q.Where("bill_batch_section_id = ?", id)
		}
	}

	// term_id
	if s := strings.TrimSpace(c.Query("term_id")); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			q = q.Where("bill_batch_term_id = ?", id)
		}
	}

	// ym=YYYY-MM
	if ym := strings.TrimSpace(c.Query("ym")); ym != "" {
		var y, m int
		if _, err := fmt.Sscanf(ym, "%d-%d", &y, &m); err == nil && y >= 2000 && y <= 2100 && m >= 1 && m <= 12 {
			q = q.Where("bill_batch_year = ? AND bill_batch_month = ?", y, m)
		}
	}

	// q: title contains
	if s := strings.TrimSpace(c.Query("q")); s != "" {
		q = q.Where("LOWER(bill_batch_title) LIKE ?", "%"+strings.ToLower(s)+"%")
	}

	// === Sorting whitelist ===
	allowedSort := map[string]string{
		"created_at": "bill_batch_created_at",
		"updated_at": "bill_batch_updated_at",
		"due_date":   "bill_batch_due_date",
		"title":      "bill_batch_title",
		"ym":         "bill_batch_year, bill_batch_month",
	}
	sortBy := strings.ToLower(strings.TrimSpace(c.Query("sort_by")))
	sortCol, ok := allowedSort[sortBy]
	if !ok {
		sortCol = allowedSort["created_at"]
	}
	dir := "DESC"
	if strings.EqualFold(strings.TrimSpace(c.Query("order")), "asc") {
		dir = "ASC"
	}
	orderClause := sortCol + " " + dir

	// === Count ===
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	// === Fetch (respect per_page=all) ===
	var rows []billing.BillBatch
	listQ := q.Order(orderClause)
	if !allMode {
		listQ = listQ.Limit(pg.PerPage).Offset(offset)
	}
	if err := listQ.Find(&rows).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	data := dto.ToBillBatchResponses(rows)

	// === Pagination untuk JsonList ===
	var pagination helper.Pagination
	if allMode {
		pagination = helper.BuildPaginationFromPage(total, 1, int(total))
	} else {
		pagination = helper.BuildPaginationFromPage(total, pg.Page, pg.PerPage)
	}

	return helper.JsonList(c, "OK", data, pagination)
}
