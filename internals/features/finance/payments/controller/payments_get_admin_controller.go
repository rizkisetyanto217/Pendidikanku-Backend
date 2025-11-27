// file: internals/features/finance/payments/controller/payment_controller.go
package controller

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	dto "madinahsalam_backend/internals/features/finance/payments/dto"
	model "madinahsalam_backend/internals/features/finance/payments/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"
)

/* =========================================================
   Admin/DKM — List payments by School
   GET /api/a/payments/list    (school_id dari token/context)
   Query:
     - view=compact|full (default: full)
========================================================= */

func (h *PaymentController) ListPaymentsBySchoolAdmin(c *fiber.Ctx) error {
	// --- 0) Ambil school_id dari TOKEN/CONTEXT (bukan dari path lagi)
	sid, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}

	// --- 1) Guard: hanya DKM/Admin untuk school ini
	if aerr := helperAuth.EnsureDKMSchool(c, sid); aerr != nil {
		return aerr
	}

	// --- 2) Filters
	q := strings.TrimSpace(c.Query("q"))
	statuses := splitCSV(c.Query("status"))
	methods := splitCSV(c.Query("method"))
	providers := splitCSV(c.Query("provider"))
	entryTypes := splitCSV(c.Query("entry_type"))
	category := strings.TrimSpace(c.Query("category"))

	// view mode: "", "compact", "full"
	view := strings.ToLower(strings.TrimSpace(c.Query("view")))

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

	// ===== Paging & sorting (pakai helper) =====
	paging := helper.ResolvePaging(c, 20, 200)
	page := paging.Page
	perPage := paging.PerPage
	offset := paging.Offset

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

	// --- 3) Query builder (base)
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

	// --- filter category dari JSONB payment_meta
	if category != "" {
		db = db.Where("payment_meta->>'fee_rule_gbk_category_snapshot' = ?", category)
	}

	if q != "" {
		ilike := "%" + q + "%"
		db = db.Where(`
			COALESCE(payment_external_id,'') ILIKE ? OR
			COALESCE(payment_gateway_reference,'') ILIKE ? OR
			COALESCE(payment_invoice_number,'') ILIKE ? OR
			COALESCE(payment_manual_reference,'') ILIKE ? OR
			COALESCE(payment_description,'') ILIKE ?
		`, ilike, ilike, ilike, ilike, ilike)
	}

	// --- 4) Count
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "count failed: "+err.Error())
	}

	// --- 5) Data
	tx := db.Order(order).Limit(perPage).Offset(offset)

	// optimisasi kolom saat view=compact
	if view == "compact" {
		tx = tx.Select([]string{
			"payment_id",
			"payment_status",
			"payment_amount_idr",
			"payment_method",
			"payment_gateway_provider",
			"payment_entry_type",

			"payment_invoice_number",
			"payment_external_id",
			"payment_gateway_reference",
			"payment_manual_reference",
			"payment_description",

			"payment_meta", // untuk ambil snapshot payer_name, student_name, dll
			"payment_created_at",
		})
	}

	var rows []model.Payment
	if err := tx.Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "query failed: "+err.Error())
	}

	// --- 6) Pagination payload (standar helper)
	pag := helper.BuildPaginationFromPage(total, page, perPage)

	// --- 7) Mapping sesuai view
	if view == "compact" {
		compact := dto.FromModelsCompact(rows)
		return helper.JsonList(c, "ok", compact, pag)
	}

	// default: full payload (PaymentResponse lama)
	data := make([]*dto.PaymentResponse, 0, len(rows))
	for i := range rows {
		data = append(data, dto.FromModel(&rows[i]))
	}

	// --- 8) Return (pakai JsonList)
	return helper.JsonList(c, "ok", data, pag)
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
