package controller

import (
	"fmt"
	"masjidku_backend/internals/features/finance/billings/dto"
	billing "masjidku_backend/internals/features/finance/billings/model"
	helper "masjidku_backend/internals/helpers"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// =======================================================
// LIST (filters + pagination)
// =======================================================

func (h *BillBatchHandler) ListBillBatches(c *fiber.Ctx) error {
	// parse pagination & sorting via helper
	p := helper.ParseFiber(c, "created_at", "desc", helper.DefaultOpts)
	offset := (p.Page - 1) * p.PerPage

	q := h.DB.Model(&billing.BillBatch{}).Where("bill_batch_deleted_at IS NULL")

	// Filters
	if s := c.Query("masjid_id"); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			q = q.Where("bill_batch_masjid_id = ?", id)
		}
	}
	if s := c.Query("class_id"); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			q = q.Where("bill_batch_class_id = ?", id)
		}
	}
	if s := c.Query("section_id"); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			q = q.Where("bill_batch_section_id = ?", id)
		}
	}
	if s := c.Query("term_id"); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			q = q.Where("bill_batch_term_id = ?", id)
		}
	}
	// ym=YYYY-MM
	if ym := c.Query("ym"); ym != "" {
		var y, m int
		if _, err := fmt.Sscanf(ym, "%d-%d", &y, &m); err == nil && y >= 2000 && y <= 2100 && m >= 1 && m <= 12 {
			q = q.Where("bill_batch_year = ? AND bill_batch_month = ?", y, m)
		}
	}
	// q: title contains
	if s := c.Query("q"); s != "" {
		q = q.Where("LOWER(bill_batch_title) LIKE ?", "%"+strings.ToLower(s)+"%")
	}

	// Sorting whitelist
	allowedSort := map[string]string{
		"created_at": "bill_batch_created_at",
		"updated_at": "bill_batch_updated_at",
		"due_date":   "bill_batch_due_date",
		"title":      "bill_batch_title",
		"ym":         "bill_batch_year, bill_batch_month",
	}
	sortCol, ok := allowedSort[p.SortBy]
	if !ok {
		sortCol = allowedSort["created_at"]
	}
	dir := "DESC"
	if strings.EqualFold(p.SortOrder, "asc") {
		dir = "ASC"
	}
	orderClause := sortCol + " " + dir

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	var rows []billing.BillBatch
	listQ := q.Order(orderClause)
	if !p.All {
		listQ = listQ.Limit(p.PerPage).Offset(offset)
	}
	if err := listQ.Find(&rows).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	data := dto.ToBillBatchResponses(rows)
	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, data, meta)
}
