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
	// Guard: owner/dkm/teacher (sesuaikan kebijakanmu)
	if !(helperAuth.IsOwner(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Akses ditolak"})
	}

	var req dto.CreateGeneralBillingRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := ctl.Validator.Struct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	// Tenant scope: pastikan masjid_id konsisten dengan context & akses
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Masjid context tidak valid"})
	}
	mid, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}
	if req.GeneralBillingMasjidID != mid {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Masjid tidak cocok dengan context"})
	}

	gb, err := req.ToModel()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if err := ctl.DB.Create(gb).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(dto.FromModelGeneralBilling(gb))
}

// ========== Patch ==========
func (ctl *GeneralBillingController) Patch(c *fiber.Ctx) error {
	if !(helperAuth.IsOwner(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Akses ditolak"})
	}

	idStr := strings.TrimSpace(c.Params("id"))
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "general_billing_id invalid"})
	}

	var gb model.GeneralBilling
	if err := model.ScopeAlive(ctl.DB).
		First(&gb, "general_billing_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Data tidak ditemukan"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Tenant guard: hanya boleh edit pada masjid yg diakses
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Masjid context tidak valid"})
	}
	mid, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}
	if gb.GeneralBillingMasjidID != mid {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Tidak boleh mengubah data tenant lain"})
	}

	var req dto.PatchGeneralBillingRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if err := req.ApplyTo(&gb); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if err := ctl.DB.Save(&gb).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(dto.FromModelGeneralBilling(&gb))
}

// ========== Delete (soft delete) ==========
func (ctl *GeneralBillingController) Delete(c *fiber.Ctx) error {
	if !(helperAuth.IsOwner(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Akses ditolak"})
	}

	idStr := strings.TrimSpace(c.Params("id"))
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "general_billing_id invalid"})
	}

	// Tenant guard via context
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Masjid context tidak valid"})
	}
	mid, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}

	// Hanya boleh menghapus record milik tenant yg sama dan yang masih alive
	tx := ctl.DB.Model(&model.GeneralBilling{}).
		Where("general_billing_id = ? AND general_billing_masjid_id = ? AND general_billing_deleted_at IS NULL", id, mid).
		Update("general_billing_deleted_at", gorm.Expr("NOW()"))

	if tx.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": tx.Error.Error()})
	}
	if tx.RowsAffected == 0 {
		// either not found or already deleted or tenant mismatch
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Data tidak ditemukan"})
	}

	return c.SendStatus(fiber.StatusNoContent) // 204
}
