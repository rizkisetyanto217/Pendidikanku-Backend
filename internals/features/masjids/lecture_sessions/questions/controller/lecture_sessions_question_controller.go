package controller

import (
	"masjidku_backend/internals/features/masjids/lecture_sessions/questions/dto"
	"masjidku_backend/internals/features/masjids/lecture_sessions/questions/model"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var validateLectureQuestion = validator.New()

type LectureSessionsQuestionController struct {
	DB *gorm.DB
}

func NewLectureSessionsQuestionController(db *gorm.DB) *LectureSessionsQuestionController {
	return &LectureSessionsQuestionController{DB: db}
}

// =============================
// ➕ Create Question
// =============================
// =============================
// ➕ Create Question
// =============================
func (ctrl *LectureSessionsQuestionController) CreateLectureSessionsQuestion(c *fiber.Ctx) error {
	var body dto.CreateLectureSessionsQuestionRequest
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}
	if err := validateLectureQuestion.Struct(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	question := model.LectureSessionsQuestionModel{
		LectureSessionsQuestion:            body.LectureSessionsQuestion,
		LectureSessionsQuestionAnswer:      body.LectureSessionsQuestionAnswer,
		LectureSessionsQuestionCorrect:     body.LectureSessionsQuestionCorrect,
		LectureSessionsQuestionExplanation: body.LectureSessionsQuestionExplanation,
		LectureSessionsQuestionQuizID:      body.LectureSessionsQuestionQuizID,
		LectureSessionsQuestionExamID:      body.LectureSessionsQuestionExamID,
	}

	if err := ctrl.DB.Create(&question).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to create question")
	}

	return c.Status(fiber.StatusCreated).JSON(dto.ToLectureSessionsQuestionDTO(question))
}

// =============================
// 📄 Get All Questions
// =============================
func (ctrl *LectureSessionsQuestionController) GetAllLectureSessionsQuestions(c *fiber.Ctx) error {
	var questions []model.LectureSessionsQuestionModel

	if err := ctrl.DB.Find(&questions).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch questions")
	}

	var response []dto.LectureSessionsQuestionDTO
	for _, q := range questions {
		response = append(response, dto.ToLectureSessionsQuestionDTO(q))
	}

	return c.JSON(response)
}

// =============================
// 🔍 Get Question by ID
// =============================
func (ctrl *LectureSessionsQuestionController) GetLectureSessionsQuestionByID(c *fiber.Ctx) error {
	id := c.Params("id")

	var question model.LectureSessionsQuestionModel
	if err := ctrl.DB.First(&question, "lecture_sessions_question_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Question not found")
	}

	return c.JSON(dto.ToLectureSessionsQuestionDTO(question))
}

// =============================
// ❌ Delete Question by ID
// =============================
func (ctrl *LectureSessionsQuestionController) DeleteLectureSessionsQuestion(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := ctrl.DB.Delete(&model.LectureSessionsQuestionModel{}, "lecture_sessions_question_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete question")
	}

	return c.SendStatus(fiber.StatusNoContent)
}
