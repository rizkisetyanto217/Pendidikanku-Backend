package controller

import (
	"log"
	"masjidku_backend/internals/features/home/advices/dto"
	"masjidku_backend/internals/features/home/advices/model"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var validateAdvice = validator.New()

type AdviceController struct {
	DB *gorm.DB
}

func NewAdviceController(db *gorm.DB) *AdviceController {
	return &AdviceController{DB: db}
}

// =======================
// ➕ Create Advice
// =======================
func (ctrl *AdviceController) CreateAdvice(c *fiber.Ctx) error {
	var body dto.CreateAdviceRequest
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}
	if err := validateAdvice.Struct(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Ambil user_id dari token (di-set oleh middleware sebelumnya)
	userID := c.Locals("user_id")
	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "User ID not found in token")
	}

	advice := model.AdviceModel{
		AdviceDescription: body.AdviceDescription,
		AdviceLectureID:   body.AdviceLectureID,
		AdviceUserID:      userIDStr,
	}

	if err := ctrl.DB.Create(&advice).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to create advice")
	}

	return c.Status(fiber.StatusCreated).JSON(dto.ToAdviceDTO(advice))
}

// =======================
// 📄 Get All Advices
// =======================
func (ctrl *AdviceController) GetAllAdvices(c *fiber.Ctx) error {
	log.Println("GetAllAdvices")
	var advices []model.AdviceModel
	if err := ctrl.DB.Find(&advices).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve advices")
	}

	var result []dto.AdviceDTO
	for _, a := range advices {
		result = append(result, dto.ToAdviceDTO(a))
	}

	return c.JSON(result)
}

// =============================
// 🔍 Get Advices by Lecture ID
// =============================
func (ctrl *AdviceController) GetAdvicesByLectureID(c *fiber.Ctx) error {
	lectureID := c.Params("lectureId")
	var advices []model.AdviceModel

	if err := ctrl.DB.Where("advice_lecture_id = ?", lectureID).Find(&advices).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch advices")
	}

	var response []dto.AdviceDTO
	for _, a := range advices {
		response = append(response, dto.ToAdviceDTO(a))
	}

	return c.JSON(response)
}

// =============================
// 🔍 Get Advices by User ID
// =============================
func (ctrl *AdviceController) GetAdvicesByUserID(c *fiber.Ctx) error {
	userID := c.Params("userId")
	var advices []model.AdviceModel

	if err := ctrl.DB.Where("advice_user_id = ?", userID).Find(&advices).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch advices")
	}

	var response []dto.AdviceDTO
	for _, a := range advices {
		response = append(response, dto.ToAdviceDTO(a))
	}

	return c.JSON(response)
}

// =============================
// 🗑️ Delete Advice by ID
// =============================
func (ctrl *AdviceController) DeleteAdvice(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := ctrl.DB.Delete(&model.AdviceModel{}, "advice_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete advice")
	}

	return c.JSON(fiber.Map{
		"message": "Advice deleted successfully",
	})
}
