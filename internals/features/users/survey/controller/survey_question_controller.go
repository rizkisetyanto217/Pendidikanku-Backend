package controller

import (
	"encoding/json"
	"masjidku_backend/internals/features/users/survey/dto"
	"masjidku_backend/internals/features/users/survey/model"
	helper "masjidku_backend/internals/helpers"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type SurveyQuestionController struct {
	DB *gorm.DB
}

func NewSurveyQuestionController(db *gorm.DB) *SurveyQuestionController {
	return &SurveyQuestionController{DB: db}
}

// ✅ GetAll: seluruh pertanyaan, urut asc by order_index
func (ctrl *SurveyQuestionController) GetAll(c *fiber.Ctx) error {
	var questions []model.SurveyQuestion
	if err := ctrl.DB.
		Order("survey_question_order_index ASC").
		Find(&questions).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to fetch questions")
	}

	// map ke DTO
	responses := make([]dto.SurveyQuestionResponse, 0, len(questions))
	for _, q := range questions {
		responses = append(responses, dto.SurveyQuestionResponse{
			SurveyQuestionID:         q.SurveyQuestionID,
			SurveyQuestionText:       q.SurveyQuestionText,
			SurveyQuestionAnswer:     q.SurveyQuestionAnswer,
			SurveyQuestionOrderIndex: q.SurveyQuestionOrderIndex,
		})
	}

	return helper.JsonList(c, responses, nil)
}

// ✅ GetByID: ambil 1 pertanyaan by ID
func (ctrl *SurveyQuestionController) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")

	var q model.SurveyQuestion
	if err := ctrl.DB.First(&q, "survey_question_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Question not found")
	}

	resp := dto.SurveyQuestionResponse{
		SurveyQuestionID:         q.SurveyQuestionID,
		SurveyQuestionText:       q.SurveyQuestionText,
		SurveyQuestionAnswer:     q.SurveyQuestionAnswer,
		SurveyQuestionOrderIndex: q.SurveyQuestionOrderIndex,
	}
	return helper.JsonOK(c, "OK", resp)
}

// ✅ Create: terima single object atau array objects
func (ctrl *SurveyQuestionController) Create(c *fiber.Ctx) error {
	body := c.Body()
	if len(body) == 0 {
		return helper.JsonError(c, fiber.StatusBadRequest, "Empty request body")
	}

	// Jika array
	if body[0] == '[' {
		var payloads []model.SurveyQuestion
		if err := json.Unmarshal(body, &payloads); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Invalid array request body")
		}
		if len(payloads) == 0 {
			return helper.JsonError(c, fiber.StatusBadRequest, "Payload array must not be empty")
		}

		// ambil order terakhir
		var maxOrder int
		if err := ctrl.DB.Model(&model.SurveyQuestion{}).
			Select("COALESCE(MAX(survey_question_order_index), 0)").
			Scan(&maxOrder).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to get max order index")
		}

		now := time.Now()
		for i := range payloads {
			payloads[i].SurveyQuestionOrderIndex = maxOrder + i + 1
			payloads[i].CreatedAt = now
			payloads[i].UpdatedAt = now
		}

		if err := ctrl.DB.Create(&payloads).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to insert questions")
		}

		// map ke DTO utk response
		responses := make([]dto.SurveyQuestionResponse, 0, len(payloads))
		for _, q := range payloads {
			responses = append(responses, dto.SurveyQuestionResponse{
				SurveyQuestionID:         q.SurveyQuestionID,
				SurveyQuestionText:       q.SurveyQuestionText,
				SurveyQuestionAnswer:     q.SurveyQuestionAnswer,
				SurveyQuestionOrderIndex: q.SurveyQuestionOrderIndex,
			})
		}

		return helper.JsonCreated(c, "Multiple questions created", responses)
	}

	// Jika single object
	var payload model.SurveyQuestion
	if err := json.Unmarshal(body, &payload); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid object request body")
	}

	var maxOrder int
	if err := ctrl.DB.Model(&model.SurveyQuestion{}).
		Select("COALESCE(MAX(survey_question_order_index), 0)").
		Scan(&maxOrder).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to get max order index")
	}

	now := time.Now()
	payload.SurveyQuestionOrderIndex = maxOrder + 1
	payload.CreatedAt = now
	payload.UpdatedAt = now

	if err := ctrl.DB.Create(&payload).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to insert question")
	}

	resp := dto.SurveyQuestionResponse{
		SurveyQuestionID:         payload.SurveyQuestionID,
		SurveyQuestionText:       payload.SurveyQuestionText,
		SurveyQuestionAnswer:     payload.SurveyQuestionAnswer,
		SurveyQuestionOrderIndex: payload.SurveyQuestionOrderIndex,
	}
	return helper.JsonCreated(c, "Single question created", resp)
}

// ✅ Update: ubah pertanyaan by ID
func (ctrl *SurveyQuestionController) Update(c *fiber.Ctx) error {
	id := c.Params("id")

	var q model.SurveyQuestion
	if err := ctrl.DB.First(&q, "survey_question_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Question not found")
	}

	var payload model.SurveyQuestion
	if err := c.BodyParser(&payload); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid request body")
	}

	// update fields yang diizinkan
	q.SurveyQuestionText = payload.SurveyQuestionText
	q.SurveyQuestionAnswer = payload.SurveyQuestionAnswer
	q.UpdatedAt = time.Now()

	if err := ctrl.DB.Save(&q).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to update question")
	}

	resp := dto.SurveyQuestionResponse{
		SurveyQuestionID:         q.SurveyQuestionID,
		SurveyQuestionText:       q.SurveyQuestionText,
		SurveyQuestionAnswer:     q.SurveyQuestionAnswer,
		SurveyQuestionOrderIndex: q.SurveyQuestionOrderIndex,
	}
	return helper.JsonUpdated(c, "Question updated", resp)
}

// ✅ Delete: hapus pertanyaan by ID
func (ctrl *SurveyQuestionController) Delete(c *fiber.Ctx) error {
	id := c.Params("id")

	// opsional: cek ada dulu
	var exists int64
	if err := ctrl.DB.Model(&model.SurveyQuestion{}).
		Where("survey_question_id = ?", id).
		Count(&exists).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to delete question")
	}
	if exists == 0 {
		return helper.JsonError(c, fiber.StatusNotFound, "Question not found")
	}

	if err := ctrl.DB.Delete(&model.SurveyQuestion{}, "survey_question_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to delete question")
	}

	return helper.JsonDeleted(c, "Question deleted successfully", fiber.Map{"id": id})
}
