// file: internals/features/finance/payments/controller/payment_controller.go
package controller

import (
	"errors"
	"strconv"
	"strings"
	"time"

	dto "madinahsalam_backend/internals/features/finance/payments/dto"
	model "madinahsalam_backend/internals/features/finance/payments/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ================= HANDLER: LIST & DETAIL PAYMENT (user / DKM) =================
//
// GET /api/u/payments/list
//
// Contoh:
//   - /api/u/payments/list
//     -> user biasa: list payment milik user ini (by payment_user_id)
//     -> DKM: list SEMUA payment di sekolah (scope=school)
//   - /api/u/payments/list?mine=true
//     -> DKM pun jadi "my payments" (by payment_user_id)
//   - /api/u/payments/list?payment-id=UUID
//     -> detail 1 payment
//   - /api/u/payments/list?status=pending&view=compact&page=1&per_page=20
//   - /api/u/payments/list?q=INV-2025&from=2025-01-01&to=2025-01-31
func (h *PaymentController) List(c *fiber.Ctx) error {
	// 1) Auth & school context
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return err
	}

	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}

	if er := helperAuth.EnsureMemberSchool(c, schoolID); er != nil {
		return er
	}

	// 1.1) Cek apakah user ini DKM/Admin sekolah
	isDKM := false
	if er := helperAuth.EnsureDKMSchool(c, schoolID); er == nil {
		isDKM = true
	}

	// Optional: paksa "my payments" walaupun DKM
	mine := strings.ToLower(strings.TrimSpace(c.Query("mine"))) == "true"

	// 2) Tentukan mode: LIST vs DETAIL (payment-id/payment_id)
	idQuery := strings.TrimSpace(c.Query("payment-id", ""))
	if idQuery == "" {
		idQuery = strings.TrimSpace(c.Query("payment_id", ""))
	}

	// ===================== MODE DETAIL =====================
	if idQuery != "" {
		pid, er := uuid.Parse(idQuery)
		if er != nil || pid == uuid.Nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "payment_id tidak valid")
		}

		var p model.PaymentModel
		if err := h.DB.WithContext(c.Context()).
			Where("payment_id = ? AND payment_school_id = ? AND payment_deleted_at IS NULL",
				pid, schoolID).
			Take(&p).Error; err != nil {

			if errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusNotFound, "payment tidak ditemukan")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "gagal mengambil payment: "+err.Error())
		}

		// Pastikan memang boleh diakses:
		//  - DKM: boleh akses semua payment di sekolah
		//  - non-DKM: hanya payment milik diri sendiri (payment_user_id)
		if !isDKM {
			if p.PaymentUserID == nil || *p.PaymentUserID != userID {
				return helper.JsonError(c, fiber.StatusForbidden, "kamu tidak berhak mengakses payment ini")
			}
		}

		return helper.JsonOK(c, "payment detail", dto.FromModel(&p))
	}

	// ===================== MODE LIST =====================

	// Filter dasar
	statuses := splitCSV(c.Query("status", ""))
	methods := splitCSV(c.Query("method"))
	providers := splitCSV(c.Query("provider"))
	entryTypes := splitCSV(c.Query("entry_type"))
	category := strings.TrimSpace(c.Query("category"))
	q := strings.TrimSpace(c.Query("q"))
	view := strings.ToLower(strings.TrimSpace(c.Query("view"))) // compact|full

	// Date range
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

	// Kompatibilitas lama: ?limit=... → override per_page
	if limitStr := strings.TrimSpace(c.Query("limit", "")); limitStr != "" {
		if lim, er := strconv.Atoi(limitStr); er == nil && lim > 0 {
			if lim > 200 {
				lim = 200
			}
			c.Context().QueryArgs().Set("per_page", strconv.Itoa(lim))
		}
	}

	// Paging standar
	paging := helper.ResolvePaging(c, 20, 200)
	page := paging.Page
	perPage := paging.PerPage
	offset := paging.Offset

	// Sorting sederhana
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

	// Base query: filter tenant + soft delete
	db := h.DB.WithContext(c.Context()).
		Model(&model.PaymentModel{}).
		Where("payment_school_id = ? AND payment_deleted_at IS NULL", schoolID)

	// Mode filter utama:
	//  - isDKM && !mine: semua payment sekolah (TANPA filter payment_user_id)
	//  - lainnya: my payments (by payment_user_id)
	if isDKM && !mine {
		// DKM mode: lihat semua payment sekolah
	} else {
		// Default: my payments by user_id
		db = db.Where("payment_user_id = ?", userID)
	}

	// Filter tanggal
	if fromPtr != nil {
		db = db.Where("payment_created_at >= ?", *fromPtr)
	}
	if toPtr != nil {
		db = db.Where("payment_created_at < ?", *toPtr)
	}

	// Filter enums
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

	// Filter category di JSONB meta (masih pakai meta untuk sekarang)
	if category != "" {
		db = db.Where("LOWER(payment_meta->>'fee_rule_gbk_category_snapshot') = LOWER(?)", category)
	}

	// Search text
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

	// Count
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal menghitung payment: "+err.Error())
	}

	// Data query
	tx := db.Order(order).Limit(perPage).Offset(offset)

	// Compact: hemat kolom
	if view == "compact" {
		tx = tx.Select([]string{
			"payment_id",
			"payment_number",
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

			"payment_meta",
			"payment_created_at",

			// snapshot payer (dipakai di PaymentCompactResponse.PayerName)
			"payment_user_name_snapshot",
			"payment_full_name_snapshot",

			// snapshot academic term (kalau nanti ditambah lagi ke header)
			// sementara: jeśli header belum punya kolom ini, hapus dari select

			// snapshot channel / VA (dipakai di PaymentCompactResponse.*VA*)
			"payment_channel_snapshot",
			"payment_bank_snapshot",
			"payment_va_number_snapshot",
			"payment_va_name_snapshot",
		})
	}

	var rows []model.PaymentModel
	if err := tx.Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal mengambil daftar payment: "+err.Error())
	}

	pag := helper.BuildPaginationFromPage(total, page, perPage)

	// Mapping sesuai view
	if view == "compact" {
		compact := dto.FromModelsCompact(rows)
		return helper.JsonList(c, "my payments", compact, pag)
	}

	full := make([]*dto.PaymentResponse, 0, len(rows))
	for i := range rows {
		full = append(full, dto.FromModel(&rows[i]))
	}

	return helper.JsonList(c, "my payments", full, pag)
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
