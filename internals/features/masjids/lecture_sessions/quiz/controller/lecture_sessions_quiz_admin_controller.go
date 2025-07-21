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

func (ctrl *LectureSessionsQuizController) CreateQuiz(c *fiber.Ctx) error {
	var body dto.CreateLectureSessionsQuizRequest
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}
	if err := validate.Struct(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Ambil masjid_id dari JWT (dari middleware)
	masjidID := c.Locals("masjid_id")
	if masjidID == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}

	quiz := model.LectureSessionsQuizModel{
		LectureSessionsQuizTitle:            body.LectureSessionsQuizTitle,
		LectureSessionsQuizDescription:      body.LectureSessionsQuizDescription,
		LectureSessionsQuizLectureSessionID: body.LectureSessionsQuizLectureSessionID,
		LectureSessionsQuizMasjidID:         masjidID.(string),
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
// 🏷️ Get Quiz By Masjid ID
// =============================
func (ctrl *LectureSessionsQuizController) GetQuizzesByMasjidID(c *fiber.Ctx) error {
	masjidID := c.Locals("masjid_id")
	if masjidID == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}

	var quizzes []model.LectureSessionsQuizModel
	if err := ctrl.DB.Where("lecture_sessions_quiz_masjid_id = ?", masjidID).Find(&quizzes).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data quiz")
	}

	var result []dto.LectureSessionsQuizDTO
	for _, quiz := range quizzes {
		result = append(result, dto.ToLectureSessionsQuizDTO(quiz))
	}

	return c.JSON(result)
}



// =============================
// ✏️ Update Quiz By ID
// =============================
func (ctrl *LectureSessionsQuizController) UpdateQuizByID(c *fiber.Ctx) error {
	id := c.Params("id")

	var body dto.UpdateLectureSessionsQuizRequest
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	var quiz model.LectureSessionsQuizModel
	if err := ctrl.DB.First(&quiz, "lecture_sessions_quiz_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Quiz tidak ditemukan")
	}

	// Partial update
	if body.LectureSessionsQuizTitle != "" {
		quiz.LectureSessionsQuizTitle = body.LectureSessionsQuizTitle
	}
	if body.LectureSessionsQuizDescription != "" {
		quiz.LectureSessionsQuizDescription = body.LectureSessionsQuizDescription
	}

	if err := ctrl.DB.Save(&quiz).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui quiz")
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
