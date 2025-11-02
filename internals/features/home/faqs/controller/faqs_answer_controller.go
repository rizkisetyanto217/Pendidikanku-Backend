package controller

import (
	"schoolku_backend/internals/features/home/faqs/dto"
	"schoolku_backend/internals/features/home/faqs/model"
	helper "schoolku_backend/internals/helpers"

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
// Create Answer
// =========================
func (ctrl *FaqAnswerController) CreateFaqAnswer(c *fiber.Ctx) error {
	var body dto.CreateFaqAnswerRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid request body")
	}

	// ambil user dari token; boleh nil jika sistem izinkan
	var answeredByPtr *string
	if uid, ok := c.Locals("user_id").(string); ok && uid != "" {
		answeredByPtr = &uid
	}

	answer := body.ToModel(answeredByPtr)

	if err := ctrl.DB.Create(&answer).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to create answer")
	}

	// preload user biar bisa isi AnsweredByName di DTO (opsional)
	if err := ctrl.DB.Preload("User").
		First(&answer, "faq_answer_id = ?", answer.FaqAnswerID).Error; err != nil {
		// kalau gagal preload, tetap balikin data utamanya
	}

	return helper.JsonCreated(c, "Answer created successfully", dto.ToFaqAnswerDTO(answer))
}

// =========================
/* Update Answer */
// =========================
func (ctrl *FaqAnswerController) UpdateFaqAnswer(c *fiber.Ctx) error {
	id := c.Params("id")

	var body dto.UpdateFaqAnswerRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid request")
	}

	var answer model.FaqAnswerModel
	if err := ctrl.DB.First(&answer, "faq_answer_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Answer not found")
	}

	body.ApplyToModel(&answer)

	if err := ctrl.DB.Save(&answer).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to update answer")
	}

	// preload user untuk DTO
	if err := ctrl.DB.Preload("User").
		First(&answer, "faq_answer_id = ?", id).Error; err != nil {
	}

	return helper.JsonUpdated(c, "Answer updated successfully", dto.ToFaqAnswerDTO(answer))
}

// =========================
/* Delete (Soft Delete) */
// =========================
func (ctrl *FaqAnswerController) DeleteFaqAnswer(c *fiber.Ctx) error {
	id := c.Params("id")

	// GORM soft delete → update kolom faq_answer_deleted_at
	if err := ctrl.DB.Delete(&model.FaqAnswerModel{}, "faq_answer_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to delete answer")
	}

	// Trigger DB kita akan re-evaluate is_answered otomatis
	return helper.JsonDeleted(c, "Answer deleted successfully", fiber.Map{"faq_answer_id": id})
}

// =========================
/* Get by ID */
// =========================
func (ctrl *FaqAnswerController) GetFaqAnswerByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var answer model.FaqAnswerModel

	if err := ctrl.DB.Preload("User").
		First(&answer, "faq_answer_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Answer not found")
	}

	return helper.JsonOK(c, "Answer fetched successfully", dto.ToFaqAnswerDTO(answer))
}
