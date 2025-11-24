// file: internals/features/finance/spp/api/student_bill_controller.go
package controller

import (
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	// ✅ DTO pakai paket SPP (bukan billings/dto)
	dto "madinahsalam_backend/internals/features/finance/billings/dto"
	// Model tetap dari billings/model
	billing "madinahsalam_backend/internals/features/finance/billings/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"
)

type StudentBillHandler struct {
	DB *gorm.DB
}

/* =========================
   List (GET /:school_id/spp/student-bills)
========================= */

func (h *StudentBillHandler) List(c *fiber.Ctx) error {
	schoolID, err := mustSchoolID(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid school_id")
	}
	// read-only access untuk semua member sekolah
	if err := helperAuth.EnsureMemberSchool(c, schoolID); err != nil {
		return err
	}

	// Paging
	pg := helper.ResolvePaging(c, 20, 200) // default 20, max 200
	allMode := strings.EqualFold(strings.TrimSpace(c.Query("per_page")), "all")

	// Sorting whitelist
	allowedSort := map[string]string{
		"created_at": "student_bill_created_at",
		"amount":     "student_bill_amount_idr",
		"status":     "student_bill_status",
		"paid_at":    "student_bill_paid_at",
		"due_date":   "student_bill_due_date", // kalau kolom ada
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

	// Base query (tenant-alive)
	q := h.DB.WithContext(c.Context()).
		Model(&billing.StudentBill{}).
		Where("student_bill_deleted_at IS NULL").
		Where("student_bill_school_id = ?", schoolID)

	// Filters umum
	if v := strings.TrimSpace(c.Query("batch_id")); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			q = q.Where("student_bill_batch_id = ?", id)
		}
	}
	if v := strings.TrimSpace(c.Query("student_id")); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			q = q.Where("student_bill_school_student_id = ?", id)
		}
	}
	if v := strings.TrimSpace(c.Query("payer_user_id")); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			q = q.Where("student_bill_payer_user_id = ?", id)
		}
	}

	// Denorm filters
	if v := strings.TrimSpace(c.Query("general_billing_kind_id")); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			q = q.Where("student_bill_general_billing_kind_id = ?", id)
		}
	}
	if v := strings.TrimSpace(c.Query("bill_code")); v != "" {
		q = q.Where("LOWER(student_bill_bill_code) = ?", strings.ToLower(v))
	}
	if v := strings.TrimSpace(c.Query("term_id")); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			q = q.Where("student_bill_term_id = ?", id)
		}
	}
	if v := strings.TrimSpace(c.Query("year")); v != "" {
		q = q.Where("student_bill_year = ?", v)
	}
	if v := strings.TrimSpace(c.Query("month")); v != "" {
		q = q.Where("student_bill_month = ?", v)
	}
	if v := strings.TrimSpace(c.Query("option_code")); v != "" {
		q = q.Where("LOWER(student_bill_option_code) = ?", strings.ToLower(v))
	}
	// has_option: periodic=false/true (periodic = option_code NULL)
	if v := strings.TrimSpace(c.Query("has_option")); v != "" { // true|false
		if strings.EqualFold(v, "true") {
			q = q.Where("student_bill_option_code IS NOT NULL")
		} else if strings.EqualFold(v, "false") {
			q = q.Where("student_bill_option_code IS NULL")
		}
	}

	// Status / amount / date
	if v := strings.TrimSpace(c.Query("status")); v != "" { // unpaid|paid|canceled
		q = q.Where("student_bill_status = ?", v)
	}
	if v := strings.TrimSpace(c.Query("paid")); v != "" { // paid=true|false
		if strings.EqualFold(v, "true") {
			q = q.Where("student_bill_paid_at IS NOT NULL")
		} else if strings.EqualFold(v, "false") {
			q = q.Where("student_bill_paid_at IS NULL")
		}
	}
	if v := c.QueryInt("amount_min"); v > 0 {
		q = q.Where("student_bill_amount_idr >= ?", v)
	}
	if v := c.QueryInt("amount_max"); v > 0 {
		q = q.Where("student_bill_amount_idr <= ?", v)
	}

	// date_from/date_to: dukung RFC3339 & YYYY-MM-DD (date_to < next-day)
	if v := strings.TrimSpace(c.Query("date_from")); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			q = q.Where("student_bill_created_at >= ?", t)
		} else if t, err := time.Parse("2006-01-02", v); err == nil {
			q = q.Where("student_bill_created_at >= ?", t)
		}
	}
	if v := strings.TrimSpace(c.Query("date_to")); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			q = q.Where("student_bill_created_at <= ?", t)
		} else if t, err := time.Parse("2006-01-02", v); err == nil {
			q = q.Where("student_bill_created_at < ?", t.AddDate(0, 0, 1))
		}
	}

	// Count
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Fetch
	var list []billing.StudentBill
	query := q.Order(orderExpr).Order("student_bill_id DESC")
	if !allMode {
		query = query.Limit(pg.Limit).Offset(pg.Offset)
	}
	if err := query.Find(&list).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	resp := dto.ToStudentBillResponses(list)

	// Pagination payload
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

	return helper.JsonList(c, "List student bills", resp, pagination)
}

/* =========================
   Create (POST /:school_id/spp/student-bills)
========================= */

func (h *StudentBillHandler) Create(c *fiber.Ctx) error {
	schoolID, err := mustSchoolID(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid school_id")
	}
	// write: staff only
	if err := helperAuth.EnsureStaffSchool(c, schoolID); err != nil {
		return err
	}

	var in dto.StudentBillCreateDTO
	if err := c.BodyParser(&in); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid json")
	}

	// Enforce tenant dari path
	in.StudentBillSchoolID = schoolID

	// Normalisasi bill_code & option
	in.StudentBillBillCode = normalizeBillCode(in.StudentBillBillCode)
	in.StudentBillOptionCode = strPtrOrNil(in.StudentBillOptionCode)

	// Periodic vs One-off validation:
	if isOneOff(in.StudentBillOptionCode) {
		// one-off: YM opsional
	} else {
		// periodic: YM wajib
		if in.StudentBillYear == nil || in.StudentBillMonth == nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "periodic student bill requires year and month")
		}
	}

	m := dto.StudentBillCreateDTOToModel(in)
	if err := h.DB.Create(&m).Error; err != nil {
		if isUniqueViolation(err) {
			// bisa dari unique per batch / partial unique periodic/oneoff
			return helper.JsonError(c, fiber.StatusConflict, "duplicate student bill for the given scope/period")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.JsonCreated(c, "created", dto.ToStudentBillResponse(m))
}

/* =========================
   Update (PATCH /:school_id/spp/student-bills/:id)
========================= */

func (h *StudentBillHandler) Update(c *fiber.Ctx) error {
	schoolID, err := mustSchoolID(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid school_id")
	}
	if err := helperAuth.EnsureStaffSchool(c, schoolID); err != nil {
		return err
	}

	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid id")
	}

	var in dto.StudentBillUpdateDTO
	if err := c.BodyParser(&in); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid json")
	}

	// Normalisasi untuk field relevan
	if in.StudentBillBillCode != nil {
		code := normalizeBillCode(*in.StudentBillBillCode)
		in.StudentBillBillCode = &code
	}
	if in.StudentBillOptionCode != nil {
		in.StudentBillOptionCode = strPtrOrNil(in.StudentBillOptionCode)
	}

	var m billing.StudentBill
	if err := h.DB.First(&m,
		"student_bill_id = ? AND student_bill_school_id = ? AND student_bill_deleted_at IS NULL",
		id, schoolID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "student_bill not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Apply partial changes
	dto.ApplyStudentBillUpdate(&m, in)

	// Re-validate periodic vs one-off setelah apply:
	if isOneOff(m.StudentBillOptionCode) {
		// one-off: YM opsional
	} else {
		// periodic: YM wajib
		if m.StudentBillYear == nil || m.StudentBillMonth == nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "periodic student bill requires year and month")
		}
	}

	if err := h.DB.Save(&m).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "duplicate student bill for the given scope/period")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.JsonUpdated(c, "updated", dto.ToStudentBillResponse(m))
}

/* =========================
   Delete (soft) — DELETE /:school_id/spp/student-bills/:id
========================= */

func (h *StudentBillHandler) Delete(c *fiber.Ctx) error {
	schoolID, err := mustSchoolID(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid school_id")
	}
	if err := helperAuth.EnsureStaffSchool(c, schoolID); err != nil {
		return err
	}

	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid id")
	}

	var m billing.StudentBill
	if err := h.DB.First(&m,
		"student_bill_id = ? AND student_bill_school_id = ? AND student_bill_deleted_at IS NULL",
		id, schoolID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "student_bill not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	if err := h.DB.Delete(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.JsonDeleted(c, "deleted", dto.ToStudentBillResponse(m))
}

/* =========================
   Status change
========================= */

// POST /:school_id/spp/student-bills/:id/cancel
func (h *StudentBillHandler) Cancel(c *fiber.Ctx) error {
	schoolID, err := mustSchoolID(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid school_id")
	}
	if err := helperAuth.EnsureStaffSchool(c, schoolID); err != nil {
		return err
	}

	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid id")
	}

	var in dto.StudentBillCancelDTO
	if err := c.BodyParser(&in); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid json")
	}

	var m billing.StudentBill
	if err := h.DB.First(&m,
		"student_bill_id = ? AND student_bill_school_id = ? AND student_bill_deleted_at IS NULL",
		id, schoolID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "student_bill not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	m.StudentBillStatus = billing.StudentBillStatusCanceled
	m.StudentBillNote = in.Note
	// optional: kosongkan paid_at saat cancel
	// m.StudentBillPaidAt = nil

	if err := h.DB.Save(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.JsonUpdated(c, "canceled", dto.ToStudentBillResponse(m))
}
