// file: internals/features/finance/general_billings/controller/general_billing_controller.go
package controller

import (
	"errors"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "masjidku_backend/internals/features/finance/general_billings/dto"
	model "masjidku_backend/internals/features/finance/general_billings/model"

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
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
func (ctl *GeneralBillingController) Create(c *fiber.Ctx) error {
	// role dasar: owner/dkm/teacher
	if !(helperAuth.IsOwner(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak")
	}

	var req dto.CreateGeneralBillingRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	if err := ctl.Validator.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Context tenant
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Masjid context tidak valid")
	}
	mid, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}

	// Guard: GLOBAL vs TENANT
	if req.GeneralBillingMasjidID == nil {
		// GLOBAL item: batasi ke Owner saja
		if !helperAuth.IsOwner(c) {
			return helper.JsonError(c, fiber.StatusForbidden, "Hanya owner yang boleh membuat billing GLOBAL")
		}
	} else {
		// TENANT item: harus cocok dengan mid context
		if *req.GeneralBillingMasjidID != mid {
			return helper.JsonError(c, fiber.StatusForbidden, "Masjid tidak cocok dengan context")
		}
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
func (ctl *GeneralBillingController) Patch(c *fiber.Ctx) error {
	if !(helperAuth.IsOwner(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak")
	}

	idStr := strings.TrimSpace(c.Params("id"))
	id, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "general_billing_id invalid")
	}

	var gb model.GeneralBilling
	if err := ctl.DB.
		Where("general_billing_id = ? AND general_billing_deleted_at IS NULL", id).
		First(&gb).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Tenant context
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Masjid context tidak valid")
	}
	mid, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}

	// Guard: GLOBAL vs TENANT pada record yg diedit
	if gb.GeneralBillingMasjidID == nil {
		// GLOBAL: hanya owner boleh edit
		if !helperAuth.IsOwner(c) {
			return helper.JsonError(c, fiber.StatusForbidden, "Hanya owner yang boleh mengubah billing GLOBAL")
		}
	} else {
		if *gb.GeneralBillingMasjidID != mid {
			return helper.JsonError(c, fiber.StatusForbidden, "Tidak boleh mengubah data tenant lain")
		}
	}

	var req dto.PatchGeneralBillingRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// (opsional) kamu bisa tambahkan guard agar PATCH tidak memindah-mindahkan tenant tanpa hak
	// misalnya, jika req.GeneralBillingMasjidID.Set == true → tolak kecuali owner, dsb.

	if err := req.ApplyTo(&gb); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	if err := ctl.DB.Save(&gb).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonOK(c, "general_billing updated", dto.FromModelGeneralBilling(&gb))
}

// ========== Delete (soft delete) ==========
func (ctl *GeneralBillingController) Delete(c *fiber.Ctx) error {
	if !(helperAuth.IsOwner(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak")
	}

	idStr := strings.TrimSpace(c.Params("id"))
	id, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "general_billing_id invalid")
	}

	// Ambil dulu record untuk cek tenant/global
	var gb model.GeneralBilling
	if err := ctl.DB.
		Where("general_billing_id = ? AND general_billing_deleted_at IS NULL", id).
		First(&gb).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Tenant context
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Masjid context tidak valid")
	}
	mid, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}

	// Guard: GLOBAL vs TENANT
	if gb.GeneralBillingMasjidID == nil {
		if !helperAuth.IsOwner(c) {
			return helper.JsonError(c, fiber.StatusForbidden, "Hanya owner yang boleh menghapus billing GLOBAL")
		}
	} else {
		if *gb.GeneralBillingMasjidID != mid {
			return helper.JsonError(c, fiber.StatusForbidden, "Tidak boleh menghapus data tenant lain")
		}
	}

	// Soft delete
	tx := ctl.DB.Model(&model.GeneralBilling{}).
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
