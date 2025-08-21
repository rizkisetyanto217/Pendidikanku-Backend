package controller

import (
	"errors"

	"masjidku_backend/internals/features/masjids/lecture_sessions/quizzes/dto"
	"masjidku_backend/internals/features/masjids/lecture_sessions/quizzes/model"
	resp "masjidku_backend/internals/helpers"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
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
		return resp.JsonError(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if err := validate.Struct(&body); err != nil {
		return resp.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Ambil masjid_id dari JWT (middleware)
	masjidIDStr, ok := c.Locals("masjid_id").(string)
	if !ok || masjidIDStr == "" {
		return resp.JsonError(c, fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	if _, err := uuid.Parse(masjidIDStr); err != nil {
		return resp.JsonError(c, fiber.StatusBadRequest, "Masjid ID tidak valid")
	}

	// Cek duplikasi: satu sesi satu kuis per masjid
	var existing model.LectureSessionsQuizModel
	err := ctrl.DB.WithContext(c.Context()).
		Where("lecture_sessions_quiz_lecture_session_id = ? AND lecture_sessions_quiz_masjid_id = ?",
			body.LectureSessionsQuizLectureSessionID, masjidIDStr).
		First(&existing).Error

	switch {
	case err == nil:
		return resp.JsonError(c, fiber.StatusConflict, "Quiz untuk sesi kajian ini sudah tersedia")
	case !errors.Is(err, gorm.ErrRecordNotFound):
		return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal memeriksa quiz yang sudah ada")
	}

	quiz := model.LectureSessionsQuizModel{
		LectureSessionsQuizTitle:            body.LectureSessionsQuizTitle,
		LectureSessionsQuizDescription:      body.LectureSessionsQuizDescription,
		LectureSessionsQuizLectureSessionID: body.LectureSessionsQuizLectureSessionID,
		LectureSessionsQuizMasjidID:         masjidIDStr,
	}

	if err := ctrl.DB.WithContext(c.Context()).Create(&quiz).Error; err != nil {
		return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat quiz")
	}

	return resp.JsonCreated(c, "Quiz created", dto.ToLectureSessionsQuizDTO(quiz))
}

// =============================
// 📄 Get All Quiz
// =============================
func (ctrl *LectureSessionsQuizController) GetAllQuizzes(c *fiber.Ctx) error {
	var quizzes []model.LectureSessionsQuizModel
	if err := ctrl.DB.WithContext(c.Context()).Find(&quizzes).Error; err != nil {
		return resp.JsonError(c, fiber.StatusInternalServerError, "Failed to fetch quizzes")
	}

	result := make([]dto.LectureSessionsQuizDTO, 0, len(quizzes))
	for _, q := range quizzes {
		result = append(result, dto.ToLectureSessionsQuizDTO(q))
	}
	return resp.JsonOK(c, "OK", result)
}

// =============================
// 🔍 Get Quiz By ID
// =============================
func (ctrl *LectureSessionsQuizController) GetQuizByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if _, err := uuid.Parse(idStr); err != nil {
	 return resp.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var quiz model.LectureSessionsQuizModel
	if err := ctrl.DB.WithContext(c.Context()).
		First(&quiz, "lecture_sessions_quiz_id = ?", idStr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp.JsonError(c, fiber.StatusNotFound, "Quiz not found")
		}
		return resp.JsonError(c, fiber.StatusInternalServerError, "Failed to get quiz")
	}

	return resp.JsonOK(c, "OK", dto.ToLectureSessionsQuizDTO(quiz))
}

// =============================
// 🏷️ Get Quiz By Masjid ID (dari token)
// =============================
func (ctrl *LectureSessionsQuizController) GetQuizzesByMasjidID(c *fiber.Ctx) error {
	masjidIDStr, ok := c.Locals("masjid_id").(string)
	if !ok || masjidIDStr == "" {
		return resp.JsonError(c, fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	if _, err := uuid.Parse(masjidIDStr); err != nil {
		return resp.JsonError(c, fiber.StatusBadRequest, "Masjid ID tidak valid")
	}

	var quizzes []model.LectureSessionsQuizModel
	if err := ctrl.DB.WithContext(c.Context()).
		Where("lecture_sessions_quiz_masjid_id = ?", masjidIDStr).
		Find(&quizzes).Error; err != nil {
		return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data quiz")
	}

	result := make([]dto.LectureSessionsQuizDTO, 0, len(quizzes))
	for _, q := range quizzes {
		result = append(result, dto.ToLectureSessionsQuizDTO(q))
	}
	return resp.JsonOK(c, "OK", result)
}

// =============================
// ✏️ Update Quiz By ID (partial)
// =============================
func (ctrl *LectureSessionsQuizController) UpdateQuizByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if _, err := uuid.Parse(idStr); err != nil {
		return resp.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var body dto.UpdateLectureSessionsQuizRequest
	if err := c.BodyParser(&body); err != nil {
		return resp.JsonError(c, fiber.StatusBadRequest, "Invalid request body")
	}

	var quiz model.LectureSessionsQuizModel
	if err := ctrl.DB.WithContext(c.Context()).
		First(&quiz, "lecture_sessions_quiz_id = ?", idStr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp.JsonError(c, fiber.StatusNotFound, "Quiz tidak ditemukan")
		}
		return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil quiz")
	}

	// Partial update
	if body.LectureSessionsQuizTitle != "" {
		quiz.LectureSessionsQuizTitle = body.LectureSessionsQuizTitle
	}
	if body.LectureSessionsQuizDescription != "" {
		quiz.LectureSessionsQuizDescription = body.LectureSessionsQuizDescription
	}

	if err := ctrl.DB.WithContext(c.Context()).Save(&quiz).Error; err != nil {
		return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui quiz")
	}

	return resp.JsonUpdated(c, "Quiz updated", dto.ToLectureSessionsQuizDTO(quiz))
}

// =============================
// ❌ Delete Quiz By ID
// =============================
func (ctrl *LectureSessionsQuizController) DeleteQuizByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if _, err := uuid.Parse(idStr); err != nil {
		return resp.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	if err := ctrl.DB.WithContext(c.Context()).
		Delete(&model.LectureSessionsQuizModel{}, "lecture_sessions_quiz_id = ?", idStr).Error; err != nil {
		return resp.JsonError(c, fiber.StatusInternalServerError, "Failed to delete quiz")
	}

	return resp.JsonDeleted(c, "Quiz deleted successfully", fiber.Map{"id": idStr})
}
