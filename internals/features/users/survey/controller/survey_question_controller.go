package controller

import (
	"encoding/json"
	"time"

	"madinahsalam_backend/internals/features/users/survey/dto"
	"madinahsalam_backend/internals/features/users/survey/model"
	helper "madinahsalam_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type SurveyQuestionController struct {
	DB *gorm.DB
}

func NewSurveyQuestionController(db *gorm.DB) *SurveyQuestionController {
	return &SurveyQuestionController{DB: db}
}

/*
	=========================================================
	  GET ALL — urut ASC by survey_question_order_index

=========================================================
*/
func (ctrl *SurveyQuestionController) GetAll(c *fiber.Ctx) error {
	var questions []model.SurveyQuestion
	if err := ctrl.DB.
		Order("survey_question_order_index ASC").
		Find(&questions).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to fetch questions")
	}

	responses := make([]dto.SurveyQuestionResponse, 0, len(questions))
	for _, q := range questions {
		responses = append(responses, dto.SurveyQuestionResponse{
			SurveyQuestionID:         q.SurveyQuestionID,
			SurveyQuestionText:       q.SurveyQuestionText,
			SurveyQuestionAnswer:     q.SurveyQuestionAnswer,
			SurveyQuestionOrderIndex: q.SurveyQuestionOrderIndex,
		})
	}

	// Tidak perlu pagination → kirim nil
	return helper.JsonList(c, "ok", responses, nil)
}

/*
	=========================================================
	  GET BY ID

=========================================================
*/
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
	return helper.JsonOK(c, "ok", resp)
}

/*
	=========================================================
	  CREATE — terima single object atau array

=========================================================
*/
func (ctrl *SurveyQuestionController) Create(c *fiber.Ctx) error {
	body := c.Body()
	if len(body) == 0 {
		return helper.JsonError(c, fiber.StatusBadRequest, "Empty request body")
	}

	// ===== Case: Array
	if body[0] == '[' {
		var payloads []model.SurveyQuestion
		if err := json.Unmarshal(body, &payloads); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Invalid array request body")
		}
		if len(payloads) == 0 {
			return helper.JsonError(c, fiber.StatusBadRequest, "Payload array must not be empty")
		}

		// Ambil order terakhir
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

		responses := make([]dto.SurveyQuestionResponse, 0, len(payloads))
		for _, q := range payloads {
			responses = append(responses, dto.SurveyQuestionResponse{
				SurveyQuestionID:         q.SurveyQuestionID,
				SurveyQuestionText:       q.SurveyQuestionText,
				SurveyQuestionAnswer:     q.SurveyQuestionAnswer,
				SurveyQuestionOrderIndex: q.SurveyQuestionOrderIndex,
			})
		}

		return helper.JsonCreated(c, "created", responses)
	}

	// ===== Case: Single object
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
	return helper.JsonCreated(c, "created", resp)
}

/*
	=========================================================
	  UPDATE

=========================================================
*/
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
	return helper.JsonUpdated(c, "updated", resp)
}

/*
	=========================================================
	  DELETE

=========================================================
*/
func (ctrl *SurveyQuestionController) Delete(c *fiber.Ctx) error {
	id := c.Params("id")

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

	return helper.JsonDeleted(c, "deleted", fiber.Map{"id": id})
}
