package controller

import (
	"log"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"masjidku_backend/internals/features/users/user/model"
	helper "masjidku_backend/internals/helpers"
)

type UserController struct {
	DB *gorm.DB
}

func NewUserController(db *gorm.DB) *UserController {
	return &UserController{DB: db}
}

// GET all users
func (uc *UserController) GetUsers(c *fiber.Ctx) error {
	var users []model.UserModel
	if err := uc.DB.Find(&users).Error; err != nil {
		log.Println("[ERROR] Failed to fetch users:", err)
		return helper.Error(c, fiber.StatusInternalServerError, "Failed to retrieve users")
	}

	// Kosongkan password semua user
	for i := range users {
		users[i].Password = ""
	}

	log.Printf("[SUCCESS] Retrieved %d users\n", len(users))
	return helper.Success(c, "Users fetched successfully", fiber.Map{
		"total": len(users),
		"users": users,
	})
}

// GET profile user by ID (dari JWT)
func (uc *UserController) GetUser(c *fiber.Ctx) error {
	userIDRaw := c.Locals("user_id")
	userIDStr, ok := userIDRaw.(string)
	if !ok {
		return helper.Error(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return helper.Error(c, fiber.StatusUnauthorized, "Invalid user ID format")
	}

	var user model.UserModel
	if err := uc.DB.First(&user, userID).Error; err != nil {
		return helper.Error(c, fiber.StatusNotFound, "User not found")
	}

	user.Password = ""
	return helper.Success(c, "User profile fetched successfully", user)
}

// POST create new user(s)
func (uc *UserController) CreateUser(c *fiber.Ctx) error {
	var singleUser model.UserModel
	var multipleUsers []model.UserModel

	if err := c.BodyParser(&multipleUsers); err == nil && len(multipleUsers) > 0 {
		if err := uc.DB.Create(&multipleUsers).Error; err != nil {
			log.Println("[ERROR] Failed to create multiple users:", err)
			return helper.Error(c, fiber.StatusInternalServerError, "Failed to create multiple users")
		}
		log.Printf("[SUCCESS] Created %d users\n", len(multipleUsers))
		return helper.SuccessWithCode(c, fiber.StatusCreated, "Users created successfully", multipleUsers)
	}

	if err := c.BodyParser(&singleUser); err != nil {
		log.Println("[ERROR] Invalid input format:", err)
		return helper.Error(c, fiber.StatusBadRequest, "Invalid input format")
	}

	if err := uc.DB.Create(&singleUser).Error; err != nil {
		log.Println("[ERROR] Failed to create user:", err)
		return helper.Error(c, fiber.StatusInternalServerError, "Failed to create user")
	}

	// Kosongkan password
	singleUser.Password = ""

	log.Printf("[SUCCESS] Created single user ID: %v\n", singleUser.ID)
	return helper.SuccessWithCode(c, fiber.StatusCreated, "User created successfully", singleUser)
}

// PUT update user by ID
type UpdateUserInput struct {
	UserName     string  `json:"user_name" validate:"required,min=3,max=50"`
	Email        string  `json:"email" validate:"required,email"`
	DonationName *string `json:"donation_name"`
	OriginalName *string `json:"original_name"`
}

func (uc *UserController) UpdateUser(c *fiber.Ctx) error {
	userIDRaw := c.Locals("user_id")
	if userIDRaw == nil {
		return helper.Error(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	userIDStr, ok := userIDRaw.(string)
	if !ok {
		return helper.Error(c, fiber.StatusInternalServerError, "Invalid user ID in token")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "Invalid UUID format")
	}

	var user model.UserModel
	if err := uc.DB.First(&user, "id = ?", userID).Error; err != nil {
		return helper.Error(c, fiber.StatusNotFound, "User not found")
	}

	var input UpdateUserInput
	if err := c.BodyParser(&input); err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "Invalid request body")
	}

	validate := validator.New()
	if err := validate.Struct(&input); err != nil {
		// Kalau error validasi, lebih baik pakai ErrorWithDetails
		return helper.ErrorWithDetails(c, fiber.StatusBadRequest, "Validation error", err.Error())
	}

	// Update fields
	user.UserName = input.UserName
	user.Email = input.Email

	if err := uc.DB.Save(&user).Error; err != nil {
		log.Println("[ERROR] Failed to update user:", err)
		return helper.Error(c, fiber.StatusInternalServerError, "Failed to update user")
	}

	log.Printf("[SUCCESS] Updated user ID: %v\n", user.ID)
	return helper.Success(c, "User updated successfully", fiber.Map{
		"id":         user.ID,
		"user_name":  user.UserName,
		"email":      user.Email,
		"updated_at": user.UpdatedAt,
	})
}

// DELETE user by ID
func (uc *UserController) DeleteUser(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := uc.DB.Delete(&model.UserModel{}, "id = ?", id).Error; err != nil {
		log.Println("[ERROR] Failed to delete user:", err)
		return helper.Error(c, fiber.StatusInternalServerError, "Failed to delete user")
	}

	log.Printf("[SUCCESS] Deleted user ID: %s\n", id)
	return helper.Success(c, "User deleted successfully", fiber.Map{
		"id": id,
	})
}
