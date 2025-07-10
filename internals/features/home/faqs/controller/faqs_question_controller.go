package controller

import (
	"masjidku_backend/internals/features/home/faqs/dto"
	"masjidku_backend/internals/features/home/faqs/model"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type FaqQuestionController struct {
	DB *gorm.DB
}

func NewFaqQuestionController(db *gorm.DB) *FaqQuestionController {
	return &FaqQuestionController{DB: db}
}

// ======================
// Create FaqQuestion
// ======================
func (ctrl *FaqQuestionController) CreateFaqQuestion(c *fiber.Ctx) error {
	var body dto.CreateFaqQuestionRequest
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "User not authenticated")
	}

	newFaq := body.ToModel(userID) // 💡 Clean

	if err := ctrl.DB.Create(&newFaq).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to create FAQ")
	}

	return c.Status(fiber.StatusCreated).JSON(dto.ToFaqQuestionDTO(newFaq))
}

// ======================
// Get All FaqQuestions
// ======================
func (ctrl *FaqQuestionController) GetAllFaqQuestions(c *fiber.Ctx) error {
	var faqs []model.FaqQuestionModel
	if err := ctrl.DB.Preload("FaqAnswers").Find(&faqs).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve FAQs")
	}

	var result []dto.FaqQuestionDTO
	for _, f := range faqs {
		result = append(result, dto.ToFaqQuestionDTO(f))
	}

	return c.JSON(result)
}

// ======================
// Get FaqQuestion by ID
// ======================
func (ctrl *FaqQuestionController) GetFaqQuestionByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var faq model.FaqQuestionModel

	if err := ctrl.DB.Preload("FaqAnswers").First(&faq, "faq_question_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "FAQ not found")
	}

	return c.JSON(dto.ToFaqQuestionDTO(faq))
}

// ======================
// Update FaqQuestion
// ======================
func (ctrl *FaqQuestionController) UpdateFaqQuestion(c *fiber.Ctx) error {
	id := c.Params("id")
	var body dto.UpdateFaqQuestionRequest

	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request")
	}

	var faq model.FaqQuestionModel
	if err := ctrl.DB.First(&faq, "faq_question_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "FAQ not found")
	}

	// Apply updates
	faq.FaqQuestionText = body.FaqQuestionText
	faq.FaqQuestionLectureID = body.FaqQuestionLectureID
	faq.FaqQuestionLectureSessionID = body.FaqQuestionLectureSessionID
	if body.FaqQuestionIsAnswered != nil {
		faq.FaqQuestionIsAnswered = *body.FaqQuestionIsAnswered
	}

	if err := ctrl.DB.Save(&faq).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to update FAQ")
	}

	return c.JSON(dto.ToFaqQuestionDTO(faq))
}

// ======================
// Delete FaqQuestion
// ======================
func (ctrl *FaqQuestionController) DeleteFaqQuestion(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := ctrl.DB.Delete(&model.FaqQuestionModel{}, "faq_question_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete FAQ")
	}
	return c.SendStatus(fiber.StatusNoContent)
}
