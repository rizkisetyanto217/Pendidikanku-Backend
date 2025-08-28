package controller

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	userdto "masjidku_backend/internals/features/users/user/dto"
	"masjidku_backend/internals/features/users/user/model"
	helper "masjidku_backend/internals/helpers"
)

type UserSelfController struct {
	DB *gorm.DB
}

func NewUserSelfController(db *gorm.DB) *UserSelfController {
	return &UserSelfController{DB: db}
}

// ==============================
// READ (SELF) — expose deleted_at
// ==============================

// GET /api/u/users/me
func (uc *UserSelfController) GetMe(c *fiber.Ctx) error {
	userID, err := helper.GetUserIDFromToken(c)
	if err != nil {
		return err // sudah fiber.NewError dgn kode yang sesuai
	}

	var user model.UserModel
	// Unscoped supaya profil tetap bisa dilihat walau soft-deleted
	if err := uc.DB.Unscoped().First(&user, "id = ?", userID).Error; err != nil {
		return helper.Error(c, fiber.StatusNotFound, "User not found")
	}
	return helper.Success(c, "User profile fetched successfully", userdto.FromModelWithDeletedAt(&user))
}

// ==============================
// UPDATE (SELF) — tolak jika soft-deleted
// ==============================

// PATCH /api/u/users/me
func (uc *UserSelfController) UpdateMe(c *fiber.Ctx) error {
	userID, err := helper.GetUserIDFromToken(c)
	if err != nil {
		return err
	}

	var user model.UserModel
	// pakai Unscoped untuk cek status deleted_at
	if err := uc.DB.Unscoped().First(&user, "id = ?", userID).Error; err != nil {
		return helper.Error(c, fiber.StatusNotFound, "User not found")
	}
	if user.DeletedAt.Valid {
		return helper.Error(c, fiber.StatusForbidden, "Akun Anda dalam status terhapus. Pulihkan akun terlebih dahulu sebelum mengubah profil.")
	}

	var input userdto.UpdateUserRequest
	if err := c.BodyParser(&input); err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "Invalid request body")
	}
	input.Normalize()

	v := validator.New()
	if err := v.Struct(&input); err != nil {
		return helper.ErrorWithDetails(c, fiber.StatusBadRequest, "Validation error", err.Error())
	}

	input.ApplyToModel(&user)

	if err := uc.DB.Save(&user).Error; err != nil {
		return helper.Error(c, fiber.StatusInternalServerError, "Failed to update user")
	}

	return helper.Success(c, "User updated successfully", userdto.FromModel(&user))
}

// ==============================
// SOFT DELETE & RESTORE (SELF)
// ==============================

// DELETE /api/u/users/me — soft delete diri sendiri
func (uc *UserSelfController) DeleteMe(c *fiber.Ctx) error {
	userID, err := helper.GetUserIDFromToken(c)
	if err != nil {
		return err
	}

	tx := uc.DB.Delete(&model.UserModel{}, "id = ?", userID)
	if tx.Error != nil {
		return helper.Error(c, fiber.StatusInternalServerError, "Failed to delete account")
	}
	if tx.RowsAffected == 0 {
		return helper.Error(c, fiber.StatusNotFound, "User not found")
	}

	return helper.Success(c, "Account deleted successfully", fiber.Map{"id": userID.String()})
}

// POST /api/u/users/me/restore — pulihkan akun sendiri
func (uc *UserSelfController) RestoreMe(c *fiber.Ctx) error {
	userID, err := helper.GetUserIDFromToken(c)
	if err != nil {
		return err
	}

	if err := uc.DB.Unscoped().
		Model(&model.UserModel{}).
		Where("id = ?", userID).
		Update("deleted_at", nil).Error; err != nil {
		return helper.Error(c, fiber.StatusInternalServerError, "Failed to restore account")
	}

	return helper.Success(c, "Account restored successfully", fiber.Map{"id": userID.String()})
}
