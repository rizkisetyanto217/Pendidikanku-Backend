// file: internals/features/finance/payments/controller/payments/payment_item_controller.go
package controller

import (
	"strings"

	dto "madinahsalam_backend/internals/features/finance/payments/dto"
	model "madinahsalam_backend/internals/features/finance/payments/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

/* =========================================================
   GET /payments/items
   Query:
     - payment_id   (optional) → filter by payment
     - student_id=me (optional) → filter by student di token
========================================================= */

func (h *PaymentItemController) ListPaymentItems(c *fiber.Ctx) error {
	// 1) resolve school dari token/context
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}

	// 2) pastikan user member school ini
	if er := helperAuth.EnsureMemberSchool(c, schoolID); er != nil {
		return er
	}

	// 3) optional: payment_id dari query
	var paymentID *uuid.UUID
	if s := strings.TrimSpace(c.Query("payment_id")); s != "" {
		id, er := uuid.Parse(s)
		if er != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "invalid payment_id")
		}
		paymentID = &id
	}

	// 4) optional: filter student_id=me
	var studentID *uuid.UUID
	studentParam := strings.TrimSpace(strings.ToLower(c.Query("student_id")))
	if studentParam == "me" || studentParam == "true" || studentParam == "1" {
		id, er := helperAuth.ResolveStudentIDFromContext(c, schoolID)
		if er != nil {
			return er // sudah balikin error rapi (unauthorized/forbidden)
		}
		studentID = &id
	}

	// 5) build query dasar
	q := h.DB.WithContext(c.Context()).
		Where("payment_item_school_id = ? AND payment_item_deleted_at IS NULL", schoolID)

	if paymentID != nil {
		q = q.Where("payment_item_payment_id = ?", *paymentID)
	}
	if studentID != nil {
		q = q.Where("payment_item_school_student_id = ?", *studentID)
	}

	var items []model.PaymentItemModel
	if err := q.
		Order("payment_item_created_at DESC, payment_item_index ASC").
		Find(&items).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal baca payment_items: "+err.Error())
	}

	return helper.JsonOK(c, "ok", dto.FromPaymentItemModels(items))
}
