package controller

import (
	"log"
	"time"

	"masjidku_backend/internals/features/users/user/model"
	helper "masjidku_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UsersProfileController struct {
	DB *gorm.DB
}

func NewUsersProfileController(db *gorm.DB) *UsersProfileController {
	return &UsersProfileController{DB: db}
}

func (upc *UsersProfileController) GetProfiles(c *fiber.Ctx) error {
	log.Println("[INFO] Fetching all user profiles")

	var profiles []model.UsersProfileModel
	if err := upc.DB.Find(&profiles).Error; err != nil {
		log.Println("[ERROR] Failed to fetch user profiles:", err)
		return helper.Error(c, fiber.StatusInternalServerError, "Failed to fetch user profiles")
	}

	return helper.Success(c, "User profiles fetched successfully", profiles)
}

func (upc *UsersProfileController) GetProfile(c *fiber.Ctx) error {
	userID := c.Locals("user_id")
	log.Println("[INFO] Fetching user profile with user_id:", userID)

	var profile model.UsersProfileModel
	if err := upc.DB.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		log.Println("[ERROR] User profile not found:", err)
		return helper.Error(c, fiber.StatusNotFound, "User profile not found")
	}

	return helper.Success(c, "User profile fetched successfully", profile)
}

func (upc *UsersProfileController) CreateProfile(c *fiber.Ctx) error {
	log.Println("[INFO] Creating or updating user profile")

	// Ambil user_id dari JWT
	userID := c.Locals("user_id")
	if userID == nil {
		log.Println("[ERROR] user_id not found in context")
		return helper.Error(c, fiber.StatusUnauthorized, "Unauthorized: no user_id")
	}

	var input model.UsersProfileModel
	if err := c.BodyParser(&input); err != nil {
		log.Println("[ERROR] Invalid request body:", err)
		return helper.Error(c, fiber.StatusBadRequest, "Invalid request format")
	}

	// Set user_id dari token ke model
	input.UserID = userID.(uuid.UUID)

	var existingProfile model.UsersProfileModel
	result := upc.DB.Where("user_id = ?", input.UserID).First(&existingProfile)

	if result.RowsAffected > 0 {
		if err := upc.DB.Model(&existingProfile).Updates(input).Error; err != nil {
			log.Println("[ERROR] Failed to update user profile:", err)
			return helper.Error(c, fiber.StatusInternalServerError, "Failed to update user profile")
		}
		log.Println("[SUCCESS] User profile updated:", input.UserID)
		return helper.Success(c, "User profile updated successfully", existingProfile)
	}

	if err := upc.DB.Create(&input).Error; err != nil {
		log.Println("[ERROR] Failed to create user profile:", err)
		return helper.Error(c, fiber.StatusInternalServerError, "Failed to create user profile")
	}

	log.Println("[SUCCESS] User profile created:", input.UserID)
	return helper.SuccessWithCode(c, fiber.StatusCreated, "User profile created successfully", input)
}

func (upc *UsersProfileController) UpdateProfile(c *fiber.Ctx) error {
	userID := c.Locals("user_id")
	log.Println("[INFO] Updating user profile with user_id:", userID)

	if userID == nil {
		return helper.Error(c, fiber.StatusUnauthorized, "Unauthorized - user_id missing")
	}

	var profile model.UsersProfileModel
	if err := upc.DB.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		log.Println("[ERROR] User profile not found:", err)
		return helper.Error(c, fiber.StatusNotFound, "User profile not found")
	}

	var input map[string]interface{}
	if err := c.BodyParser(&input); err != nil {
		log.Println("[ERROR] Invalid request body:", err)
		return helper.Error(c, fiber.StatusBadRequest, "Invalid request format")
	}

	// Jangan izinkan user_id diganti
	delete(input, "user_id")

	// Tambahkan updated_at manual jika kamu pakai timestamp
	input["updated_at"] = time.Now()

	if err := upc.DB.Model(&profile).Updates(input).Error; err != nil {
		log.Println("[ERROR] Failed to update user profile:", err)
		return helper.Error(c, fiber.StatusInternalServerError, "Failed to update user profile")
	}

	return helper.Success(c, "User profile updated successfully", profile)
}

func (upc *UsersProfileController) DeleteProfile(c *fiber.Ctx) error {
	userID := c.Locals("user_id")
	log.Println("[INFO] Deleting user profile with user_id:", userID)

	var profile model.UsersProfileModel
	if err := upc.DB.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		log.Println("[ERROR] User profile not found:", err)
		return helper.Error(c, fiber.StatusNotFound, "User profile not found")
	}

	if err := upc.DB.Delete(&profile).Error; err != nil {
		log.Println("[ERROR] Failed to delete user profile:", err)
		return helper.Error(c, fiber.StatusInternalServerError, "Failed to delete user profile")
	}

	return helper.Success(c, "User profile deleted successfully", nil)
}
