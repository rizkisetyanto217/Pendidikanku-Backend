// file: internals/features/finance/general_billings/controller/user_general_billing_controller.go
package controller

import (
	"errors"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "masjidku_backend/internals/features/finance/general_billings/dto"
	model "masjidku_backend/internals/features/finance/general_billings/model"
	helper "masjidku_backend/internals/helpers" // <- pastikan path-nya sesuai lokasi Json* helper kamu
)

/* ========================================================
   Controller
======================================================== */

type UserGeneralBillingController struct {
	DB        *gorm.DB
	Validator *validator.Validate
}

func NewUserGeneralBillingController(db *gorm.DB) *UserGeneralBillingController {
	return &UserGeneralBillingController{
		DB:        db,
		Validator: validator.New(),
	}
}

/* ========================================================
   Helpers
======================================================== */

func parseUUIDParam(c *fiber.Ctx, name string) (uuid.UUID, error) {
	val := c.Params(name)
	id, err := uuid.Parse(val)
	if err != nil {
		return uuid.Nil, helper.JsonError(c, fiber.StatusBadRequest, "invalid "+name)
	}
	return id, nil
}

func isPgUniqueErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "duplicate key value violates unique constraint")
}
func isPgFKErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "violates foreign key constraint")
}


/* ========================================================
   Handlers
======================================================== */

// POST /finance/user-general-billings
func (ctl *UserGeneralBillingController) Create(c *fiber.Ctx) error {
	var req dto.CreateUserGeneralBillingRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid JSON body")
	}
	if err := ctl.Validator.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	if err := req.Validate(); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	m := req.ToModel()
	now := time.Now()
	m.UserGeneralBillingCreatedAt = now
	m.UserGeneralBillingUpdatedAt = now

	if err := ctl.DB.Create(&m).Error; err != nil {
		switch {
		case isPgUniqueErr(err):
			return helper.JsonError(c, fiber.StatusConflict, "duplicate: billing already exists for the same student or payer")
		case isPgFKErr(err):
			return helper.JsonError(c, fiber.StatusBadRequest, "foreign key not found")
		default:
			return helper.JsonError(c, fiber.StatusInternalServerError, "failed to create: "+err.Error())
		}
	}

	return helper.JsonCreated(c, "created", dto.FromModelUserGeneralBilling(m))
}


// PATCH /finance/user-general-billings/:id
func (ctl *UserGeneralBillingController) Patch(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}

	var m model.UserGeneralBilling
	if err := ctl.DB.First(&m, "user_general_billing_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	var p dto.PatchUserGeneralBillingRequest
	if err := c.BodyParser(&p); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid JSON body")
	}

	changed := p.Apply(&m)
	if !changed {
		// tidak ada perubahan — tetap kembalikan state saat ini
		return helper.JsonOK(c, "no changes", dto.FromModelUserGeneralBilling(m))
	}

	m.UserGeneralBillingUpdatedAt = time.Now()

	// Validasi bisnis setelah apply
	if err := p.ValidateAfterApply(m); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	if err := ctl.DB.Save(&m).Error; err != nil {
		switch {
		case isPgUniqueErr(err):
			return helper.JsonError(c, fiber.StatusConflict, "conflict: another record already exists for that billing with same student/payer")
		case isPgFKErr(err):
			return helper.JsonError(c, fiber.StatusBadRequest, "foreign key not found")
		default:
			return helper.JsonError(c, fiber.StatusInternalServerError, "failed to save: "+err.Error())
		}
	}

	return helper.JsonUpdated(c, "updated", dto.FromModelUserGeneralBilling(m))
}

// DELETE /finance/user-general-billings/:id  (soft delete)
func (ctl *UserGeneralBillingController) Delete(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	var m model.UserGeneralBilling
	if err := ctl.DB.First(&m, "user_general_billing_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	if err := ctl.DB.Delete(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonDeleted(c, "deleted", fiber.Map{"id": id})
}
