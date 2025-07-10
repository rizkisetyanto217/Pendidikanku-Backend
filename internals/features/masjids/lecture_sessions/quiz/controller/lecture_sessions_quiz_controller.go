package controller

import (
	"masjidku_backend/internals/features/masjids/lecture_sessions/quiz/dto"
	"masjidku_backend/internals/features/masjids/lecture_sessions/quiz/model"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type LectureSessionsQuizController struct {
	DB *gorm.DB
}

func NewLectureSessionsQuizController(db *gorm.DB) *LectureSessionsQuizController {
	return &LectureSessionsQuizController{DB: db}
}

var validate = validator.New()

// =============================
// ➕ Create Quiz
// =============================
func (ctrl *LectureSessionsQuizController) CreateQuiz(c *fiber.Ctx) error {
	var body dto.CreateLectureSessionsQuizRequest
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}
	if err := validate.Struct(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	quiz := model.LectureSessionsQuizModel{
		LectureSessionsQuizTitle:            body.LectureSessionsQuizTitle,
		LectureSessionsQuizDescription:      body.LectureSessionsQuizDescription,
		LectureSessionsQuizLectureSessionID: body.LectureSessionsQuizLectureSessionID,
	}

	if err := ctrl.DB.Create(&quiz).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to create quiz")
	}

	return c.Status(fiber.StatusCreated).JSON(dto.ToLectureSessionsQuizDTO(quiz))
}

// =============================
// 📄 Get All Quiz
// =============================
func (ctrl *LectureSessionsQuizController) GetAllQuizzes(c *fiber.Ctx) error {
	var quizzes []model.LectureSessionsQuizModel

	if err := ctrl.DB.Find(&quizzes).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch quizzes")
	}

	var result []dto.LectureSessionsQuizDTO
	for _, quiz := range quizzes {
		result = append(result, dto.ToLectureSessionsQuizDTO(quiz))
	}

	return c.JSON(result)
}

// =============================
// 🔍 Get Quiz By ID
// =============================
func (ctrl *LectureSessionsQuizController) GetQuizByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var quiz model.LectureSessionsQuizModel

	if err := ctrl.DB.First(&quiz, "lecture_sessions_quiz_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Quiz not found")
	}

	return c.JSON(dto.ToLectureSessionsQuizDTO(quiz))
}

// =============================
// ❌ Delete Quiz By ID
// =============================
func (ctrl *LectureSessionsQuizController) DeleteQuizByID(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := ctrl.DB.Delete(&model.LectureSessionsQuizModel{}, "lecture_sessions_quiz_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete quiz")
	}

	return c.JSON(fiber.Map{
		"message": "Quiz deleted successfully",
	})
}
