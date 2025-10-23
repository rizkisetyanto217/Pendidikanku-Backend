// File: internals/features/finance/spp/api/bill_batch_controller.go
package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	// ganti sesuai struktur projectmu
	"masjidku_backend/internals/features/finance/billings/dto"
	billing "masjidku_backend/internals/features/finance/billings/model"
	helper "masjidku_backend/internals/helpers"
)

// =======================================================
// BOOTSTRAP
// =======================================================

type BillBatchHandler struct {
	DB *gorm.DB
}

// =======================================================
// HELPERS
// =======================================================

func parseUUIDParam(c *fiber.Ctx, name string) (uuid.UUID, error) {
	idStr := c.Params(name)
	return uuid.Parse(idStr)
}

func xorValid(classID, sectionID *uuid.UUID) bool {
	return (classID != nil && sectionID == nil) || (classID == nil && sectionID != nil)
}

func isUniqueViolation(err error) bool {
	// deteksi sederhana utk unique index (Postgres)
	return err != nil &&
		(strings.Contains(err.Error(), "duplicate key value") ||
			strings.Contains(err.Error(), "unique constraint"))
}

// =======================================================
// CREATE
// =======================================================

func (h *BillBatchHandler) CreateBillBatch(c *fiber.Ctx) error {
	var in dto.BillBatchCreateDTO
	if err := c.BodyParser(&in); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "invalid json")
	}

	// XOR guard: class vs section
	if !xorValid(in.BillBatchClassID, in.BillBatchSectionID) {
		return helper.JsonError(c, http.StatusBadRequest, "exactly one of bill_batch_class_id or bill_batch_section_id must be set")
	}

	m := dto.BillBatchCreateDTOToModel(in)

	if err := h.DB.Create(&m).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, http.StatusConflict, "duplicate bill batch for the given scope and period")
		}
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	return helper.JsonCreated(c, "bill batch created", dto.ToBillBatchResponse(m))
}

// =======================================================
// UPDATE (partial)
// =======================================================

func (h *BillBatchHandler) UpdateBillBatch(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "invalid id")
	}

	var in dto.BillBatchUpdateDTO
	if err := c.BodyParser(&in); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "invalid json")
	}

	var m billing.BillBatch
	if err := h.DB.First(&m, "bill_batch_id = ? AND bill_batch_deleted_at IS NULL", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "bill batch not found")
		}
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	if err := dto.ApplyBillBatchUpdate(&m, in); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	if err := h.DB.Save(&m).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, http.StatusConflict, "duplicate bill batch for the given scope and period")
		}
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	return helper.JsonOK(c, "bill batch updated", dto.ToBillBatchResponse(m))
}

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

// =======================================================
// DELETE (soft delete)
// =======================================================

func (h *BillBatchHandler) DeleteBillBatch(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "invalid id")
	}

	err = h.DB.Transaction(func(tx *gorm.DB) error {
		var m billing.BillBatch
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&m, "bill_batch_id = ? AND bill_batch_deleted_at IS NULL", id).Error; err != nil {
			return err
		}
		now := time.Now()
		m.BillBatchDeletedAt = gorm.DeletedAt{Time: now, Valid: true}
		return tx.Save(&m).Error
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "bill batch not found")
		}
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	return helper.JsonDeleted(c, "bill batch deleted", fiber.Map{"bill_batch_id": id})
}
