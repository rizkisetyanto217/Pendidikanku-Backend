package controller

import (
	"masjidku_backend/internals/features/home/faqs/dto"
	"masjidku_backend/internals/features/home/faqs/model"
	helper "masjidku_backend/internals/helpers"

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
// Buat Jawaban Baru (trx)
// =========================
func (ctrl *FaqAnswerController) CreateFaqAnswer(c *fiber.Ctx) error {
	var body dto.CreateFaqAnswerRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid request body")
	}

	answeredBy, ok := c.Locals("user_id").(string)
	if !ok || answeredBy == "" {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	answer := body.ToModel(answeredBy)

	// Transaksi: simpan answer + set pertanyaan menjadi answered
	if err := ctrl.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&answer).Error; err != nil {
			return err
		}
		res := tx.Model(&model.FaqQuestionModel{}).
			Where("faq_question_id = ?", body.FaqAnswerQuestionID).
			Update("faq_question_is_answered", true)
		if res.Error != nil {
			return res.Error
		}
		// Opsional: jika question tidak ditemukan (RowsAffected == 0), silakan pilih mau error atau tidak.
		return nil
	}); err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to create answer")
	}

	return helper.JsonCreated(c, "Answer created successfully", dto.ToFaqAnswerDTO(answer))
}

// =========================
// Update Jawaban
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

	answer.FaqAnswerText = body.FaqAnswerText

	if err := ctrl.DB.Save(&answer).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to update answer")
	}

	return helper.JsonUpdated(c, "Answer updated successfully", dto.ToFaqAnswerDTO(answer))
}

// =========================
// Hapus Jawaban
// =========================
func (ctrl *FaqAnswerController) DeleteFaqAnswer(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := ctrl.DB.Delete(&model.FaqAnswerModel{}, "faq_answer_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to delete answer")
	}

	// 200 + body agar konsisten dengan helper
	return helper.JsonDeleted(c, "Answer deleted successfully", fiber.Map{"faq_answer_id": id})
}

// =========================
/* Detail Jawaban */
// =========================
func (ctrl *FaqAnswerController) GetFaqAnswerByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var answer model.FaqAnswerModel

	if err := ctrl.DB.First(&answer, "faq_answer_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Answer not found")
	}

	return helper.JsonOK(c, "Answer fetched successfully", dto.ToFaqAnswerDTO(answer))
}
