package controller

import (
	"masjidku_backend/internals/features/home/questionnaires/dto"
	"masjidku_backend/internals/features/home/questionnaires/model"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type QuestionnaireQuestionController struct {
	DB *gorm.DB
}

func NewQuestionnaireQuestionController(db *gorm.DB) *QuestionnaireQuestionController {
	return &QuestionnaireQuestionController{DB: db}
}

var validate = validator.New()

// =============================
// ➕ Create Question
// =============================
func (ctrl *QuestionnaireQuestionController) CreateQuestion(c *fiber.Ctx) error {
	var req dto.CreateQuestionnaireQuestionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}
	if err := validate.Struct(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	question := dto.ToQuestionnaireQuestionModel(req)
	if err := ctrl.DB.Create(&question).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to create question")
	}

	return c.Status(fiber.StatusCreated).JSON(dto.ToQuestionnaireQuestionDTO(question))
}

// =============================
// 📄 Get All Questions
// =============================
func (ctrl *QuestionnaireQuestionController) GetAllQuestions(c *fiber.Ctx) error {
	var questions []model.QuestionnaireQuestionModel
	if err := ctrl.DB.Find(&questions).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve questions")
	}

	var response []dto.QuestionnaireQuestionDTO
	for _, q := range questions {
		response = append(response, dto.ToQuestionnaireQuestionDTO(q))
	}

	return c.JSON(response)
}

// =============================
// 🔍 Get Questions by Scope (1=general, 2=event, 3=lecture)
// =============================
func (ctrl *QuestionnaireQuestionController) GetQuestionsByScope(c *fiber.Ctx) error {
	scope := c.Params("scope") // expected: "1", "2", "3"

	var questions []model.QuestionnaireQuestionModel
	if err := ctrl.DB.Where("questionnaire_question_scope = ?", scope).Find(&questions).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve questions by scope")
	}

	var response []dto.QuestionnaireQuestionDTO
	for _, q := range questions {
		response = append(response, dto.ToQuestionnaireQuestionDTO(q))
	}

	return c.JSON(response)
}
