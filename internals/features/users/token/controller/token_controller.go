package controller

import (
	"log"
	"masjidku_backend/internals/features/users/token/model"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type TokenController struct {
	DB *gorm.DB
}

func NewTokenController(db *gorm.DB) *TokenController {
	return &TokenController{DB: db}
}

// ✅ Get all tokens
func (ctrl *TokenController) GetAll(c *fiber.Ctx) error {
	var tokens []model.Token
	if err := ctrl.DB.Order("id ASC").Find(&tokens).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch tokens"})
	}
	return c.JSON(tokens)
}

// ✅ Get token by ID
func (ctrl *TokenController) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var token model.Token
	if err := ctrl.DB.First(&token, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Token not found"})
	}
	return c.JSON(token)
}

// ✅ Create new token
func (ctrl *TokenController) Create(c *fiber.Ctx) error {
	var payload model.Token
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	log.Println("[DEBUG] Payload token:", payload)

	if err := ctrl.DB.Create(&payload).Error; err != nil {
		log.Println("[ERROR] Gagal menyimpan token:", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to create token"})
	}

	return c.Status(201).JSON(payload)
}

// ✅ Update token
func (ctrl *TokenController) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	var token model.Token
	if err := ctrl.DB.First(&token, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Token not found"})
	}

	var payload model.Token
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := ctrl.DB.Save(&token).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to update token"})
	}

	return c.JSON(token)
}

// ✅ Delete token
func (ctrl *TokenController) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := ctrl.DB.Delete(&model.Token{}, id).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to delete token"})
	}
	return c.JSON(fiber.Map{"message": "Token deleted successfully"})
}
