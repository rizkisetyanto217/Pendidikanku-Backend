package controller

import (
	"masjidku_backend/internals/features/masjids/lecture_sessions/questions/dto"
	"masjidku_backend/internals/features/masjids/lecture_sessions/questions/model"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var validateUserQuestion = validator.New()

type LectureSessionsUserQuestionController struct {
	DB *gorm.DB
}

func NewLectureSessionsUserQuestionController(db *gorm.DB) *LectureSessionsUserQuestionController {
	return &LectureSessionsUserQuestionController{DB: db}
}

// =============================
// ➕ Create
// =============================
func (ctrl *LectureSessionsUserQuestionController) CreateLectureSessionsUserQuestion(c *fiber.Ctx) error {
	var body dto.CreateLectureSessionsUserQuestionRequest
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}
	if err := validateUserQuestion.Struct(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	question := model.LectureSessionsUserQuestionModel{
		LectureSessionsUserQuestionAnswer:     body.LectureSessionsUserQuestionAnswer,
		LectureSessionsUserQuestionIsCorrect:  body.LectureSessionsUserQuestionIsCorrect,
		LectureSessionsUserQuestionQuestionID: body.LectureSessionsUserQuestionQuestionID,
	}

	if err := ctrl.DB.Create(&question).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to create user question answer")
	}

	return c.Status(fiber.StatusCreated).JSON(dto.ToLectureSessionsUserQuestionDTO(question))
}

// =============================
// 📄 Get All User Question Answers
// =============================
func (ctrl *LectureSessionsUserQuestionController) GetAllLectureSessionsUserQuestions(c *fiber.Ctx) error {
	var records []model.LectureSessionsUserQuestionModel
	if err := ctrl.DB.Find(&records).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve data")
	}

	var response []dto.LectureSessionsUserQuestionDTO
	for _, r := range records {
		response = append(response, dto.ToLectureSessionsUserQuestionDTO(r))
	}
	return c.JSON(response)
}

// =============================
// 🔍 Get by Question ID
// =============================
func (ctrl *LectureSessionsUserQuestionController) GetByQuestionID(c *fiber.Ctx) error {
	questionID := c.Params("question_id")
	var records []model.LectureSessionsUserQuestionModel

	if err := ctrl.DB.Where("lecture_sessions_user_question_question_id = ?", questionID).Find(&records).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve data by question ID")
	}

	var response []dto.LectureSessionsUserQuestionDTO
	for _, r := range records {
		response = append(response, dto.ToLectureSessionsUserQuestionDTO(r))
	}
	return c.JSON(response)
}

// =============================
// 🗑️ Delete by ID
// =============================
func (ctrl *LectureSessionsUserQuestionController) DeleteByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := ctrl.DB.Delete(&model.LectureSessionsUserQuestionModel{}, "lecture_sessions_user_question_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete record")
	}
	return c.JSON(fiber.Map{"message": "Deleted successfully"})
}
