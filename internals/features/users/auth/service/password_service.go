package service

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	authHelper "masjidku_backend/internals/features/users/auth/helper"
	authRepo "masjidku_backend/internals/features/users/auth/repository"
	helper "masjidku_backend/internals/helpers"
)

// ========================== RESET PASSWORD ==========================
func ResetPassword(db *gorm.DB, c *fiber.Ctx) error {
	var input struct {
		Email       string `json:"email"`
		NewPassword string `json:"new_password"`
	}
	if err := c.BodyParser(&input); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid request format")
	}

	// 🔹 Validasi format email dan password
	if err := authHelper.ValidateResetPassword(input.Email, input.NewPassword); err != nil {
		return helper.JsonError(c, fiber.StatusUnprocessableEntity, err.Error()) // 422 untuk validasi
	}

	// 🔹 Cari user
	user, err := authRepo.FindUserByEmail(db, input.Email)
	if err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "User not found")
	}

	// 🔹 Hash password baru
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to hash password")
	}

	// 🔹 Update password
	if err := authRepo.UpdateUserPassword(db, user.ID, string(hashedPassword)); err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to update password")
	}

	return helper.JsonUpdated(c, "Password reset successfully", nil)
}

// ========================== CHANGE PASSWORD ==========================
// ========================== CHANGE PASSWORD ==========================
func ChangePassword(db *gorm.DB, c *fiber.Ctx) error {
	var input struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := c.BodyParser(&input); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid input format")
	}

	// Ambil user_id dari Locals dengan aman
	v := c.Locals("user_id")
	userIDStr, ok := v.(string)
	if !ok || userIDStr == "" {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Unauthorized")
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Invalid user id")
	}

	// Ambil user
	user, err := authRepo.FindUserByID(db, userID)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "User not found")
	}

	// Cek password lama
	if err := authHelper.CheckPasswordHash(user.Password, input.CurrentPassword); err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Current password incorrect")
	}

	// Hash password baru
	newHash, err := authHelper.HashPassword(input.NewPassword)
	if err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to hash new password")
	}

	// Update password
	if err := authRepo.UpdateUserPassword(db, userID, newHash); err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to update password")
	}

	return helper.JsonUpdated(c, "Password changed successfully", nil)
}

