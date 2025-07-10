package service

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	authHelper "masjidku_backend/internals/features/users/auth/helper"
	authRepo "masjidku_backend/internals/features/users/auth/repository"
	helpers "masjidku_backend/internals/helpers"
)

// ========================== RESET PASSWORD ==========================
func ResetPassword(db *gorm.DB, c *fiber.Ctx) error {
	var input struct {
		Email       string `json:"email"`
		NewPassword string `json:"new_password"`
	}

	if err := c.BodyParser(&input); err != nil {
		return helpers.Error(c, fiber.StatusBadRequest, "Invalid request format")
	}

	// 🔹 Validasi format email dan password
	if err := authHelper.ValidateResetPassword(input.Email, input.NewPassword); err != nil {
		return helpers.Error(c, fiber.StatusBadRequest, err.Error())
	}

	// 🔹 Cari user dari repository
	user, err := authRepo.FindUserByEmail(db, input.Email)
	if err != nil {
		return helpers.Error(c, fiber.StatusNotFound, "User not found")
	}

	// 🔹 Hash password baru
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return helpers.Error(c, fiber.StatusInternalServerError, "Failed to hash password")
	}

	// 🔹 Update password lewat repository
	if err := authRepo.UpdateUserPassword(db, user.ID, string(hashedPassword)); err != nil {
		return helpers.Error(c, fiber.StatusInternalServerError, "Failed to update password")
	}

	return helpers.Success(c, "Password reset successfully", nil)
}

// ========================== CHANGE PASSWORD ==========================
func ChangePassword(db *gorm.DB, c *fiber.Ctx) error {
	var input struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := c.BodyParser(&input); err != nil {
		return helpers.Error(c, fiber.StatusBadRequest, "Invalid input format")
	}

	userIDStr := c.Locals("user_id").(string)
	userID, _ := uuid.Parse(userIDStr)

	user, err := authRepo.FindUserByID(db, userID)
	if err != nil {
		return helpers.Error(c, fiber.StatusUnauthorized, "User not found")
	}

	if err := authHelper.CheckPasswordHash(user.Password, input.CurrentPassword); err != nil {
		return helpers.Error(c, fiber.StatusUnauthorized, "Current password incorrect")
	}

	newHash, err := authHelper.HashPassword(input.NewPassword)
	if err != nil {
		return helpers.Error(c, fiber.StatusInternalServerError, "Failed to hash new password")
	}

	if err := authRepo.UpdateUserPassword(db, userID, newHash); err != nil {
		return helpers.Error(c, fiber.StatusInternalServerError, "Failed to update password")
	}

	return helpers.Success(c, "Password changed successfully", nil)
}
