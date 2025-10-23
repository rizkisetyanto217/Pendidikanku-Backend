// file: internals/features/finance/spp/api/student_bill_controller.go
package api

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	// ganti path sesuai projectmu
	"masjidku_backend/internals/features/finance/billings/dto"
	billing "masjidku_backend/internals/features/finance/billings/model"
	helper "masjidku_backend/internals/helpers"
)

type StudentBillHandler struct {
	DB *gorm.DB
}

func buildOrderClause(p helper.Params) string {
	// whitelist sortable keys → kolom fisik
	allowed := map[string]string{
		"created_at": "student_bill_created_at",
		"updated_at": "student_bill_updated_at",
		"amount":     "student_bill_amount_idr",
		"status":     "student_bill_status",
		"paid_at":    "student_bill_paid_at",
	}
	col, ok := allowed[strings.ToLower(p.SortBy)]
	if !ok {
		col = allowed["created_at"]
	}
	dir := "DESC"
	if strings.ToLower(p.SortOrder) == "asc" {
		dir = "ASC"
	}
	return fmt.Sprintf("%s %s", col, dir)
}

// -----------------------------------------
// List (GET /student-bills)
// Query filters (opsional):
// - masjid_id, batch_id, student_id, payer_user_id
// - option_code, status
// - paid: true|false
// - amount_min, amount_max (int)
// - date_from, date_to (filter created_at)
// - sort_by (created_at|updated_at|amount|status|paid_at), order (asc|desc)
// - page, per_page
// -----------------------------------------
func (h *StudentBillHandler) List(c *fiber.Ctx) error {
	p := helper.ParseFiber(c, "created_at", "desc", helper.DefaultOpts)

	q := h.DB.Model(&billing.StudentBill{}).
		Where("student_bill_deleted_at IS NULL")

	if v := c.Query("masjid_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			q = q.Where("student_bill_masjid_id = ?", id)
		}
	}
	if v := c.Query("batch_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			q = q.Where("student_bill_batch_id = ?", id)
		}
	}
	if v := c.Query("student_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			q = q.Where("student_bill_masjid_student_id = ?", id)
		}
	}
	if v := c.Query("payer_user_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			q = q.Where("student_bill_payer_user_id = ?", id)
		}
	}
	if v := c.Query("option_code"); v != "" {
		q = q.Where("LOWER(student_bill_option_code) = ?", strings.ToLower(v))
	}
	if v := c.Query("status"); v != "" {
		// unpaid|paid|canceled
		q = q.Where("student_bill_status = ?", v)
	}
	// paid=true/false → terjemahkan ke paid_at NULL / NOT NULL
	if v := c.Query("paid"); v != "" {
		if strings.EqualFold(v, "true") {
			q = q.Where("student_bill_paid_at IS NOT NULL")
		} else if strings.EqualFold(v, "false") {
			q = q.Where("student_bill_paid_at IS NULL")
		}
	}

	// amount range
	if v := c.QueryInt("amount_min"); v > 0 {
		q = q.Where("student_bill_amount_idr >= ?", v)
	}
	if v := c.QueryInt("amount_max"); v > 0 {
		q = q.Where("student_bill_amount_idr <= ?", v)
	}

	// date range (created_at)
	if v := c.Query("date_from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			q = q.Where("student_bill_created_at >= ?", t)
		}
	}
	if v := c.Query("date_to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			q = q.Where("student_bill_created_at <= ?", t)
		}
	}

	// count
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// data
	var list []billing.StudentBill
	if err := q.
		Order(buildOrderClause(p)).
		Limit(p.Limit()).
		Offset(p.Offset()).
		Find(&list).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	resp := dto.ToStudentBillResponses(list)
	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, resp, meta)
}

// -----------------------------------------
// Create (POST /student-bills)
// -----------------------------------------
func (h *StudentBillHandler) Create(c *fiber.Ctx) error {
	var in dto.StudentBillCreateDTO
	if err := c.BodyParser(&in); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid json")
	}
	m := dto.StudentBillCreateDTOToModel(in)
	if err := h.DB.Create(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.JsonCreated(c, "created", dto.ToStudentBillResponse(m))
}

// -----------------------------------------
// Update (PATCH /student-bills/:id)
// -----------------------------------------
func (h *StudentBillHandler) Update(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid id")
	}
	var in dto.StudentBillUpdateDTO
	if err := c.BodyParser(&in); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid json")
	}
	var m billing.StudentBill
	if err := h.DB.First(&m, "student_bill_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "student_bill not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	dto.ApplyStudentBillUpdate(&m, in)
	if err := h.DB.Save(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.JsonUpdated(c, "updated", dto.ToStudentBillResponse(m))
}

// -----------------------------------------
// Delete (DELETE /student-bills/:id) — soft delete
// -----------------------------------------
func (h *StudentBillHandler) Delete(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid id")
	}
	var m billing.StudentBill
	if err := h.DB.First(&m, "student_bill_id = ?", id).Error; err != nil {
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

// -----------------------------------------
// Status: Mark Paid (POST /student-bills/:id/mark-paid)
// -----------------------------------------
func (h *StudentBillHandler) MarkPaid(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid id")
	}
	var in dto.StudentBillMarkPaidDTO
	if err := c.BodyParser(&in); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid json")
	}

	var m billing.StudentBill
	if err := h.DB.First(&m, "student_bill_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "student_bill not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	now := time.Now()
	if in.PaidAt == nil {
		in.PaidAt = &now
	}
	m.StudentBillStatus = billing.StudentBillStatusPaid
	m.StudentBillPaidAt = in.PaidAt
	m.StudentBillNote = in.Note

	if err := h.DB.Save(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.JsonUpdated(c, "marked as paid", dto.ToStudentBillResponse(m))
}

// -----------------------------------------
// Status: Mark Unpaid (POST /student-bills/:id/mark-unpaid)
// -----------------------------------------
func (h *StudentBillHandler) MarkUnpaid(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid id")
	}
	var in dto.StudentBillMarkUnpaidDTO
	if err := c.BodyParser(&in); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid json")
	}

	var m billing.StudentBill
	if err := h.DB.First(&m, "student_bill_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "student_bill not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	m.StudentBillStatus = billing.StudentBillStatusUnpaid
	m.StudentBillPaidAt = nil
	m.StudentBillNote = in.Note

	if err := h.DB.Save(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.JsonUpdated(c, "marked as unpaid", dto.ToStudentBillResponse(m))
}

// -----------------------------------------
// Status: Cancel (POST /student-bills/:id/cancel)
// -----------------------------------------
func (h *StudentBillHandler) Cancel(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid id")
	}
	var in dto.StudentBillCancelDTO
	if err := c.BodyParser(&in); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid json")
	}

	var m billing.StudentBill
	if err := h.DB.First(&m, "student_bill_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "student_bill not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	m.StudentBillStatus = billing.StudentBillStatusCanceled
	m.StudentBillNote = in.Note

	if err := h.DB.Save(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.JsonUpdated(c, "canceled", dto.ToStudentBillResponse(m))
}
