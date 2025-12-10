// file: internals/features/finance/general_billings/controller/general_billing_controller.go
package controller

import (
	"errors"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "madinahsalam_backend/internals/features/finance/general_billings/dto"
	model "madinahsalam_backend/internals/features/finance/general_billings/model"

	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"
)

type GeneralBillingController struct {
	DB        *gorm.DB
	Validator *validator.Validate
}

func NewGeneralBillingController(db *gorm.DB) *GeneralBillingController {
	return &GeneralBillingController{
		DB:        db,
		Validator: validator.New(),
	}
}

// ========== Create ==========
// POST /api/a/general-billings
func (ctl *GeneralBillingController) Create(c *fiber.Ctx) error {
	// 1) Ambil school_id dari token / context
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err // sudah JsonError di dalam helper
	}
	if schoolID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "school context not found in token")
	}

	// 2) Hanya DKM/Admin per-school yang diizinkan
	if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
		return err
	}

	var req dto.CreateGeneralBillingRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// 3) Paksa pakai schoolID dari token (abaikan body)
	req.GeneralBillingSchoolID = schoolID

	if err := ctl.Validator.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	gb, err := req.ToModel()
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	if err := ctl.DB.Create(gb).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonCreated(c, "general_billing created", dto.FromModelGeneralBilling(gb))
}

// ========== Patch ==========
// PATCH /api/a/general-billings/:id
func (ctl *GeneralBillingController) Patch(c *fiber.Ctx) error {
	idStr := strings.TrimSpace(c.Params("id"))
	id, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "general_billing_id invalid")
	}

	// Ambil record dulu (buat cek tenant + existence)
	var gb model.GeneralBillingModel
	if err := ctl.DB.
		Where("general_billing_id = ? AND general_billing_deleted_at IS NULL", id).
		First(&gb).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Ambil school_id dari token
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}
	if schoolID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "school context not found in token")
	}

	// Hanya DKM/Admin school ini yang boleh
	if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
		return err
	}

	// Guard tenant: record harus milik tenant di context
	if gb.GeneralBillingSchoolID != schoolID {
		return helper.JsonError(c, fiber.StatusForbidden, "Tidak boleh mengubah data tenant lain")
	}

	var req dto.PatchGeneralBillingRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	if err := req.ApplyTo(&gb); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	if err := ctl.DB.Save(&gb).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonOK(c, "general_billing updated", dto.FromModelGeneralBilling(&gb))
}

// ========== Delete (soft delete) ==========
// DELETE /api/a/general-billings/:id
func (ctl *GeneralBillingController) Delete(c *fiber.Ctx) error {
	idStr := strings.TrimSpace(c.Params("id"))
	id, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "general_billing_id invalid")
	}

	// Ambil dulu record untuk cek tenant
	var gb model.GeneralBillingModel
	if err := ctl.DB.
		Where("general_billing_id = ? AND general_billing_deleted_at IS NULL", id).
		First(&gb).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Ambil school_id dari token
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}
	if schoolID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "school context not found in token")
	}

	// Hanya DKM/Admin school ini yang boleh
	if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
		return err
	}

	// Guard tenant
	if gb.GeneralBillingSchoolID != schoolID {
		return helper.JsonError(c, fiber.StatusForbidden, "Tidak boleh menghapus data tenant lain")
	}

	// Soft delete
	tx := ctl.DB.Model(&model.GeneralBillingModel{}).
		Where("general_billing_id = ? AND general_billing_deleted_at IS NULL", id).
		Update("general_billing_deleted_at", gorm.Expr("NOW()"))
	if tx.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, tx.Error.Error())
	}
	if tx.RowsAffected == 0 {
		return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
	}

	return helper.JsonDeleted(c, "general_billing deleted", fiber.Map{"general_billing_id": id})
}
