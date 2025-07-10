package controller

import (
	"masjidku_backend/internals/features/home/faqs/dto"
	"masjidku_backend/internals/features/home/faqs/model"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type FaqAnswerController struct {
	DB *gorm.DB
}

func NewFaqAnswerController(db *gorm.DB) *FaqAnswerController {
	return &FaqAnswerController{DB: db}
}

// =========================
// Buat Jawaban Baru
// =========================
func (ctrl *FaqAnswerController) CreateFaqAnswer(c *fiber.Ctx) error {
	var body dto.CreateFaqAnswerRequest
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	// Ambil user ID dari token
	answeredBy, ok := c.Locals("user_id").(string)
	if !ok || answeredBy == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	answer := body.ToModel(answeredBy)

	// Simpan jawaban
	if err := ctrl.DB.Create(&answer).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to create answer")
	}

	// Update pertanyaan jadi is_answered = true
	ctrl.DB.Model(&model.FaqQuestionModel{}).
		Where("faq_question_id = ?", body.FaqAnswerQuestionID).
		Update("faq_question_is_answered", true)

	return c.Status(fiber.StatusCreated).JSON(dto.ToFaqAnswerDTO(answer))
}

// =========================
// Update Jawaban
// =========================
func (ctrl *FaqAnswerController) UpdateFaqAnswer(c *fiber.Ctx) error {
	id := c.Params("id")
	var body dto.UpdateFaqAnswerRequest

	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request")
	}

	var answer model.FaqAnswerModel
	if err := ctrl.DB.First(&answer, "faq_answer_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Answer not found")
	}

	answer.FaqAnswerText = body.FaqAnswerText

	if err := ctrl.DB.Save(&answer).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to update answer")
	}

	return c.JSON(dto.ToFaqAnswerDTO(answer))
}

// =========================
// Hapus Jawaban
// =========================
func (ctrl *FaqAnswerController) DeleteFaqAnswer(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := ctrl.DB.Delete(&model.FaqAnswerModel{}, "faq_answer_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete answer")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// =========================
// Detail Jawaban (Opsional)
// =========================
func (ctrl *FaqAnswerController) GetFaqAnswerByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var answer model.FaqAnswerModel

	if err := ctrl.DB.First(&answer, "faq_answer_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Answer not found")
	}

	return c.JSON(dto.ToFaqAnswerDTO(answer))
}
