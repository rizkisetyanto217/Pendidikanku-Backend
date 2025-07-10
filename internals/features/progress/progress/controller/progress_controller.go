package controller

import (
	"log"

	"masjidku_backend/internals/features/progress/progress/model"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserProgressController struct {
	DB *gorm.DB
}

func NewUserProgressController(db *gorm.DB) *UserProgressController {
	return &UserProgressController{DB: db}
}

// GET /api/user-progress
func (ctrl *UserProgressController) GetByUserID(c *fiber.Ctx) error {
	userIDRaw := c.Locals("user_id")
	if userIDRaw == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized: token tidak valid",
		})
	}

	// Handle UUID format
	var userID uuid.UUID
	switch v := userIDRaw.(type) {
	case uuid.UUID:
		userID = v
	case string:
		parsed, err := uuid.Parse(v)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "user_id tidak valid dalam token",
			})
		}
		userID = parsed
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "user_id tidak dikenali",
		})
	}

	var progress model.UserProgress
	if err := ctrl.DB.Where("user_progress_user_id = ?", userID).First(&progress).Error; err != nil {
		log.Println("[ERROR] Gagal mengambil user_progress:", err)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Data progres user tidak ditemukan",
		})
	}

	return c.JSON(fiber.Map{
		"data": progress,
	})
}
