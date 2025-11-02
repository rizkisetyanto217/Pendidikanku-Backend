package controller

import (
	dto "schoolku_backend/internals/features/finance/general_billings/dto"
	model "schoolku_backend/internals/features/finance/general_billings/model"
	helper "schoolku_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
)

// GET /finance/user-general-billings
func (ctl *UserGeneralBillingController) List(c *fiber.Ctx) error {
	var q dto.ListUserGeneralBillingQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid query")
	}

	// ===== pagination & sorting via helper =====
	// allowed sort keys -> mapping ke kolom DB yang aman
	allowedSort := map[string]string{
		"created_at": "user_general_billing_created_at",
		"updated_at": "user_general_billing_updated_at",
		"amount":     "user_general_billing_amount_idr",
		"status":     "user_general_billing_status",
		"paid_at":    "user_general_billing_paid_at",
	}

	// default sort: newest first by created_at
	p := helper.ParseFiber(c, "created_at", "desc", helper.DefaultOpts)

	orderClause, err := p.SafeOrderClause(allowedSort, "created_at")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid sort field")
	}

	tx := ctl.DB.Model(&model.UserGeneralBilling{})

	// ===== Filters =====
	if q.SchoolID != nil {
		tx = tx.Where("user_general_billing_school_id = ?", *q.SchoolID)
	}
	if q.BillingID != nil {
		tx = tx.Where("user_general_billing_billing_id = ?", *q.BillingID)
	}
	if q.SchoolStudentID != nil {
		tx = tx.Where("user_general_billing_school_student_id = ?", *q.SchoolStudentID)
	}
	if q.PayerUserID != nil {
		tx = tx.Where("user_general_billing_payer_user_id = ?", *q.PayerUserID)
	}
	if q.Status != nil && *q.Status != "" {
		tx = tx.Where("user_general_billing_status = ?", *q.Status)
	}

	// ===== Count total (sebelum Limit/Offset) =====
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// ===== Query data dengan sorting & paging =====
	var rows []model.UserGeneralBilling
	qry := tx.Order(orderClause)
	if !p.All { // per_page=all -> skip limit/offset (akan dibatasi AllHardCap oleh ParseFiber)
		qry = qry.Offset(p.Offset()).Limit(p.Limit())
	}

	if err := qry.Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// ===== Map ke DTO =====
	out := make([]dto.UserGeneralBillingResponse, 0, len(rows))
	for _, m := range rows {
		out = append(out, dto.FromModelUserGeneralBilling(m))
	}

	// ===== Build meta =====
	meta := helper.BuildMeta(total, p)

	return helper.JsonList(c, out, meta)
}
