package controller

import (
	"errors"
	"strconv"
	"strings"

	dto "madinahsalam_backend/internals/features/finance/payments/dto"
	model "madinahsalam_backend/internals/features/finance/payments/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ================= HANDLER: LIST & DETAIL PAYMENT USER (satu controller) =================
//
// GET /payments
// Contoh:
//   - /payments                          -> list semua payment user ini
//   - /payments?payment-id=UUID          -> detail 1 payment
//   - /payments?payment_id=UUID          -> (fallback alternatif, opsional)
//   - /payments?status=pending&limit=10  -> list dengan filter
func (h *PaymentController) MyPayments(c *fiber.Ctx) error {
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

	// 2) Tentukan mode: LIST vs DETAIL
	//    pakai query ?payment-id=... (utama), atau ?payment_id=... (fallback)
	idQuery := strings.TrimSpace(c.Query("payment-id", ""))
	if idQuery == "" {
		idQuery = strings.TrimSpace(c.Query("payment_id", ""))
	}

	// ---- MODE DETAIL: kalau ada payment-id / payment_id di query ----
	if idQuery != "" {
		pid, er := uuid.Parse(idQuery)
		if er != nil || pid == uuid.Nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "payment_id tidak valid")
		}

		var p model.Payment
		if err := h.DB.WithContext(c.Context()).
			Where("payment_id = ? AND payment_school_id = ? AND payment_deleted_at IS NULL",
				pid, schoolID).
			Take(&p).Error; err != nil {

			if errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusNotFound, "payment tidak ditemukan")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "gagal mengambil payment: "+err.Error())
		}

		// Pastikan memang milik user ini
		if p.PaymentUserID == nil || *p.PaymentUserID != userID {
			return helper.JsonError(c, fiber.StatusForbidden, "kamu tidak berhak mengakses payment ini")
		}

		return helper.JsonOK(c, "payment detail", dto.FromModel(&p))
	}

	// ---- MODE LIST: tidak ada payment-id / payment_id → list by user_id ----
	status := strings.TrimSpace(c.Query("status", "")) // optional: pending/paid/dll
	limitStr := c.Query("limit", "20")
	limit, er := strconv.Atoi(limitStr)
	if er != nil || limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	q := h.DB.WithContext(c.Context()).
		Where("payment_school_id = ? AND payment_user_id = ? AND payment_deleted_at IS NULL",
			schoolID, userID).
		Order("payment_created_at DESC").
		Limit(limit)

	if status != "" {
		q = q.Where("payment_status = ?", strings.ToLower(status))
	}

	var rows []model.Payment
	if err := q.Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal mengambil daftar payment: "+err.Error())
	}

	out := make([]any, 0, len(rows))
	for i := range rows {
		out = append(out, dto.FromModel(&rows[i]))
	}

	return helper.JsonOK(c, "my payments", out)
}
