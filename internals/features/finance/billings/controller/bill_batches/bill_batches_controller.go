// file: internals/features/finance/spp/api/bill_batch_controller.go
package controller

import (
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	dto "madinahsalam_backend/internals/features/finance/billings/dto"
	sppmodel "madinahsalam_backend/internals/features/finance/billings/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"
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
	return err != nil &&
		(strings.Contains(err.Error(), "duplicate key value") ||
			strings.Contains(err.Error(), "unique constraint"))
}

func isOneOff(optionCode *string) bool {
	return optionCode != nil && strings.TrimSpace(*optionCode) != ""
}

func normalizeBillCode(code string) string {
	code = strings.TrimSpace(code)
	if code == "" {
		return "SPP"
	}
	return code
}

func strPtrOrNil(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

// =======================================================
// CREATE (hanya buat batch; school_id dari token/context)
// POST /api/a/spp/bill-batches
// =======================================================

func (h *BillBatchHandler) CreateBillBatch(c *fiber.Ctx) error {
	// 🔐 Ambil school_id dari token/context
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}
	// 🔐 staff guard (teacher/dkm/admin/bendahara)
	if err := helperAuth.EnsureStaffSchool(c, schoolID); err != nil {
		return err
	}

	var in dto.BillBatchCreateDTO
	if err := c.BodyParser(&in); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid json")
	}

	// override dari context
	in.BillBatchSchoolID = schoolID
	in.BillBatchBillCode = normalizeBillCode(in.BillBatchBillCode)
	in.BillBatchOptionCode = strPtrOrNil(in.BillBatchOptionCode)

	// XOR guard
	if !xorValid(in.BillBatchClassID, in.BillBatchSectionID) {
		return helper.JsonError(c, fiber.StatusBadRequest, "exactly one of bill_batch_class_id or bill_batch_section_id must be set")
	}

	// Periodic vs One-off validation:
	if isOneOff(in.BillBatchOptionCode) {
		// one-off: YM opsional (boleh nil)
	} else {
		// periodic: YM wajib
		if in.BillBatchMonth == nil || in.BillBatchYear == nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "periodic batch requires bill_batch_month and bill_batch_year")
		}
	}

	m := dto.BillBatchCreateDTOToModel(in)

	if err := h.DB.Create(&m).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "duplicate bill batch for the given scope and period")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonCreated(c, "bill batch created", dto.ToBillBatchResponse(m))
}

// =======================================================
// UPDATE (partial; tenant-guard)
// PATCH /api/a/spp/bill-batches/:id
// =======================================================

func (h *BillBatchHandler) UpdateBillBatch(c *fiber.Ctx) error {
	// 🔐 Ambil school_id dari token/context
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}
	// 🔐 staff guard
	if err := helperAuth.EnsureStaffSchool(c, schoolID); err != nil {
		return err
	}

	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid id")
	}

	var in dto.BillBatchUpdateDTO
	if err := c.BodyParser(&in); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid json")
	}

	var m sppmodel.BillBatchModel
	if err := h.DB.First(
		&m,
		"bill_batch_id = ? AND bill_batch_school_id = ? AND bill_batch_deleted_at IS NULL",
		id, schoolID,
	).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "bill batch not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	if err := dto.ApplyBillBatchUpdate(&m, in); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Re-validate periodic vs one-off setelah apply:
	if isOneOff(m.BillBatchOptionCode) {
		// one-off: YM opsional (no-op)
	} else {
		// periodic: YM wajib & valid
		if m.BillBatchMonth == nil || m.BillBatchYear == nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "periodic batch requires bill_batch_month and bill_batch_year")
		}
	}

	if err := h.DB.Save(&m).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "duplicate bill batch for the given scope and period")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonOK(c, "bill batch updated", dto.ToBillBatchResponse(m))
}

// =======================================================
// DELETE (soft delete; tenant-scoped)
// DELETE /api/a/spp/bill-batches/:id
// =======================================================

func (h *BillBatchHandler) DeleteBillBatch(c *fiber.Ctx) error {
	// 🔐 Ambil school_id dari token/context
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}
	// 🔐 staff guard
	if err := helperAuth.EnsureStaffSchool(c, schoolID); err != nil {
		return err
	}

	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid id")
	}

	err = h.DB.Transaction(func(tx *gorm.DB) error {
		var m sppmodel.BillBatchModel
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&m,
				"bill_batch_id = ? AND bill_batch_school_id = ? AND bill_batch_deleted_at IS NULL",
				id, schoolID,
			).Error; err != nil {
			return err
		}
		now := time.Now()
		m.BillBatchDeletedAt = gorm.DeletedAt{Time: now, Valid: true}
		return tx.Save(&m).Error
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "bill batch not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonDeleted(c, "bill batch deleted", fiber.Map{"bill_batch_id": id})
}
