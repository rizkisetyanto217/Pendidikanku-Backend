package controller

import (
	"strings"

	dto "madinahsalam_backend/internals/features/finance/general_billings/dto"
	model "madinahsalam_backend/internals/features/finance/general_billings/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GET /api/a/:school_id/user-general-billings
func (ctl *UserGeneralBillingController) List(c *fiber.Ctx) error {
	// ===== Resolve school context: token > active-school > path =====
	var schoolID uuid.UUID
	if id, err := helperAuth.GetSchoolIDFromTokenPreferTeacher(c); err == nil && id != uuid.Nil {
		schoolID = id
	} else if id, err := helperAuth.GetActiveSchoolID(c); err == nil && id != uuid.Nil {
		schoolID = id
	} else {
		sid, err := helperAuth.ParseSchoolIDFromPath(c)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "school context not found")
		}
		schoolID = sid
	}

	// ===== Guard: hanya staff (teacher/dkm/admin/bendahara) =====
	if err := helperAuth.EnsureStaffSchool(c, schoolID); err != nil {
		return err
	}
	c.Locals("__school_guard_ok", schoolID.String())

	// ===== Query DTO dari query params =====
	var q dto.ListUserGeneralBillingQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid query")
	}

	// ===== Pagination =====
	pg := helper.ResolvePaging(c, 20, 200) // default 20, max 200
	perPageRaw := strings.ToLower(strings.TrimSpace(c.Query("per_page")))
	allMode := perPageRaw == "all"

	// ===== Sorting whitelist =====
	allowedSort := map[string]string{
		"created_at": "user_general_billing_created_at",
		"updated_at": "user_general_billing_updated_at",
		"amount":     "user_general_billing_amount_idr",
		"status":     "user_general_billing_status",
		"paid_at":    "user_general_billing_paid_at",
	}
	sortBy := strings.ToLower(strings.TrimSpace(c.Query("sort_by", "created_at")))
	col, ok := allowedSort[sortBy]
	if !ok {
		col = allowedSort["created_at"]
	}
	dir := "DESC"
	if strings.EqualFold(strings.TrimSpace(c.Query("order")), "asc") {
		dir = "ASC"
	}
	orderExpr := col + " " + dir

	// ===== Base query: tenant-scoped =====
	tx := ctl.DB.WithContext(c.Context()).
		Model(&model.UserGeneralBilling{}).
		Where("user_general_billing_school_id = ?", schoolID)

	// ===== Filters =====
	// school_id dari query diabaikan; selalu pakai schoolID dari context

	if q.BillingID != nil {
		tx = tx.Where("user_general_billing_billing_id = ?", *q.BillingID)
	}
	if q.SchoolStudentID != nil {
		tx = tx.Where("user_general_billing_school_student_id = ?", *q.SchoolStudentID)
	}
	if q.PayerUserID != nil {
		tx = tx.Where("user_general_billing_payer_user_id = ?", *q.PayerUserID)
	}
	if q.Status != nil && strings.TrimSpace(*q.Status) != "" {
		tx = tx.Where("user_general_billing_status = ?", strings.TrimSpace(*q.Status))
	}

	// ===== Count total =====
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// ===== Data + sorting + paging =====
	var rows []model.UserGeneralBilling
	qry := tx.
		Order(orderExpr).
		Order("user_general_billing_id DESC") // tie-breaker stabil

	if !allMode {
		qry = qry.Offset(pg.Offset).Limit(pg.Limit)
	}

	if err := qry.Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// ===== Map ke DTO =====
	out := make([]dto.UserGeneralBillingResponse, 0, len(rows))
	for _, m := range rows {
		out = append(out, dto.FromModelUserGeneralBilling(m))
	}

	// ===== Pagination meta =====
	var pagination helper.Pagination
	if allMode {
		per := int(total)
		if per <= 0 {
			per = 1
		}
		pagination = helper.BuildPaginationFromPage(total, 1, per)
	} else {
		pagination = helper.BuildPaginationFromPage(total, pg.Page, pg.PerPage)
	}

	// ===== JSON response =====
	return helper.JsonList(c, "List user general billings", out, pagination)
}
