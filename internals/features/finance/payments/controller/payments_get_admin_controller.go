// file: internals/features/finance/payments/controller/payment_controller.go
package controller

import (
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	dto "schoolku_backend/internals/features/finance/payments/dto"
	model "schoolku_backend/internals/features/finance/payments/model"
	helperAuth "schoolku_backend/internals/helpers/auth"
)

/* =========================================================
   Admin/DKM — List payments by School
   GET /api/a/:school_id/payments
========================================================= */

func (h *PaymentController) ListPaymentsBySchoolAdmin(c *fiber.Ctx) error {
	// --- 0) Ambil school_id dari PATH
	sid, err := helperAuth.ParseSchoolIDFromPath(c)
	if err != nil {
		return err
	}

	// --- 1) Guard: hanya DKM/Admin
	if aerr := helperAuth.EnsureDKMSchool(c, sid); aerr != nil {
		return aerr
	}

	// --- 2) Filters
	q := strings.TrimSpace(c.Query("q"))
	statuses := splitCSV(c.Query("status"))
	methods := splitCSV(c.Query("method"))
	providers := splitCSV(c.Query("provider"))
	entryTypes := splitCSV(c.Query("entry_type"))

	var fromPtr, toPtr *time.Time
	const dFmt = "2006-01-02"
	if fs := strings.TrimSpace(c.Query("from")); fs != "" {
		if t, e := time.Parse(dFmt, fs); e == nil {
			fromPtr = &t
		}
	}
	if ts := strings.TrimSpace(c.Query("to")); ts != "" {
		if t, e := time.Parse(dFmt, ts); e == nil {
			t = t.Add(24 * time.Hour) // inklusif
			toPtr = &t
		}
	}

	// ===== Paging & sorting =====
	page := clampInt(parseIntDefault(c.Query("page"), 1), 1, 1_000_000)
	perPage := clampInt(parseIntDefault(c.Query("per_page"), 20), 1, 200)

	// fallback kompatibilitas: limit/offset
	if limStr := strings.TrimSpace(c.Query("limit")); limStr != "" {
		if lim := parseIntDefault(limStr, perPage); lim > 0 {
			perPage = clampInt(lim, 1, 200)
		}
	}
	if offStr := strings.TrimSpace(c.Query("offset")); offStr != "" {
		if off := parseIntDefault(offStr, 0); off >= 0 {
			page = off/perPage + 1
		}
	}
	offset := (page - 1) * perPage

	sort := strings.ToLower(strings.TrimSpace(c.Query("sort")))
	order := "payment_created_at DESC"
	switch sort {
	case "created_at_asc":
		order = "payment_created_at ASC"
	case "amount_desc":
		order = "payment_amount_idr DESC, payment_created_at DESC"
	case "amount_asc":
		order = "payment_amount_idr ASC, payment_created_at DESC"
	}

	// --- 3) Query builder
	db := h.DB.WithContext(c.Context()).Model(&model.Payment{}).
		Where("payment_deleted_at IS NULL").
		Where("payment_school_id = ?", sid)

	if fromPtr != nil {
		db = db.Where("payment_created_at >= ?", *fromPtr)
	}
	if toPtr != nil {
		db = db.Where("payment_created_at < ?", *toPtr)
	}
	if len(statuses) > 0 {
		db = db.Where("payment_status IN (?)", statuses)
	}
	if len(methods) > 0 {
		db = db.Where("payment_method IN (?)", methods)
	}
	if len(providers) > 0 {
		db = db.Where("payment_gateway_provider IN (?)", providers)
	}
	if len(entryTypes) > 0 {
		db = db.Where("payment_entry_type IN (?)", entryTypes)
	}

	if q != "" {
		ilike := "%" + q + "%"
		db = db.Where(`
			COALESCE(payment_external_id,'') ILIKE ? OR
			COALESCE(payment_gateway_reference,'') ILIKE ? OR
			COALESCE(invoice_number,'') ILIKE ? OR
			COALESCE(payment_manual_reference,'') ILIKE ? OR
			COALESCE(payment_description,'') ILIKE ?
		`, ilike, ilike, ilike, ilike, ilike)
	}

	// --- 4) Count & data
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "count failed: " + err.Error(),
		})
	}

	var rows []model.Payment
	if err := db.Order(order).Limit(perPage).Offset(offset).Find(&rows).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "query failed: " + err.Error(),
		})
	}

	// --- 5) Map → DTO
	data := make([]*dto.PaymentResponse, 0, len(rows))
	for i := range rows {
		data = append(data, dto.FromModel(&rows[i]))
	}

	// --- 6) Pagination payload
	totalPages := int((total + int64(perPage) - 1) / int64(perPage))
	if totalPages == 0 {
		totalPages = 1
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	// --- 7) Return -> message, data, pagination sejajar
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "ok",
		"data":    data,
		"pagination": fiber.Map{
			"page":        page,
			"per_page":    perPage,
			"total":       total,
			"total_pages": totalPages,
			"has_next":    hasNext,
			"has_prev":    hasPrev,
		},
	})
}

/* ============== small utils ============== */

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return def
}
