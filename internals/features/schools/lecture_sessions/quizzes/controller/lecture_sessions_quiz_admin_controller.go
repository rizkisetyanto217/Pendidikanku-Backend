package controller

import (
	"errors"

	"schoolku_backend/internals/features/schools/lecture_sessions/quizzes/dto"
	"schoolku_backend/internals/features/schools/lecture_sessions/quizzes/model"
	resp "schoolku_backend/internals/helpers"

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

	// Ambil school_id dari JWT (middleware)
	schoolIDStr, ok := c.Locals("school_id").(string)
	if !ok || schoolIDStr == "" {
		return resp.JsonError(c, fiber.StatusUnauthorized, "School ID tidak ditemukan di token")
	}
	if _, err := uuid.Parse(schoolIDStr); err != nil {
		return resp.JsonError(c, fiber.StatusBadRequest, "School ID tidak valid")
	}

	// Cek duplikasi: satu sesi satu kuis per school (baris hidup saja; GORM exclude soft-deleted)
	var existing model.LectureSessionsQuizModel
	err := ctrl.DB.WithContext(c.Context()).
		Where("lecture_sessions_quiz_lecture_session_id = ? AND lecture_sessions_quiz_school_id = ?",
			body.LectureSessionsQuizLectureSessionID, schoolIDStr).
		First(&existing).Error
	switch {
	case err == nil:
		return resp.JsonError(c, fiber.StatusConflict, "Quiz untuk sesi kajian ini sudah tersedia")
	case !errors.Is(err, gorm.ErrRecordNotFound):
		return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal memeriksa quiz yang sudah ada")
	}

	quiz := body.ToModel(schoolIDStr)
	if err := ctrl.DB.WithContext(c.Context()).Create(&quiz).Error; err != nil {
		// tanpa package tambahan: kirim pesan umum
		return resp.JsonError(c, fiber.StatusConflict, "Data bertentangan dengan aturan unik (mungkin judul/sesi duplikat)")
	}

	return resp.JsonCreated(c, "Quiz created", dto.ToLectureSessionsQuizDTO(*quiz))
}

// =============================
// 📄 Get All Quiz (baris hidup saja)
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
// 🏷️ Get Quiz By School ID (dari token)
// =============================
func (ctrl *LectureSessionsQuizController) GetQuizzesBySchoolID(c *fiber.Ctx) error {
	schoolIDStr, ok := c.Locals("school_id").(string)
	if !ok || schoolIDStr == "" {
		return resp.JsonError(c, fiber.StatusUnauthorized, "School ID tidak ditemukan di token")
	}
	if _, err := uuid.Parse(schoolIDStr); err != nil {
		return resp.JsonError(c, fiber.StatusBadRequest, "School ID tidak valid")
	}

	var quizzes []model.LectureSessionsQuizModel
	if err := ctrl.DB.WithContext(c.Context()).
		Where("lecture_sessions_quiz_school_id = ?", schoolIDStr).
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
	if err := validate.Struct(&body); err != nil {
		return resp.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var quiz model.LectureSessionsQuizModel
	if err := ctrl.DB.WithContext(c.Context()).
		First(&quiz, "lecture_sessions_quiz_id = ?", idStr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp.JsonError(c, fiber.StatusNotFound, "Quiz tidak ditemukan")
		}
		return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil quiz")
	}

	// Partial update (pointer-aware)
	if body.LectureSessionsQuizTitle != nil {
		quiz.LectureSessionsQuizTitle = *body.LectureSessionsQuizTitle
	}
	if body.LectureSessionsQuizDescription != nil {
		quiz.LectureSessionsQuizDescription = body.LectureSessionsQuizDescription // bisa nil utk clear
	}

	if err := ctrl.DB.WithContext(c.Context()).Save(&quiz).Error; err != nil {
		// tanpa package tambahan: pesan umum
		return resp.JsonError(c, fiber.StatusConflict, "Gagal menyimpan: kemungkinan melanggar aturan unik")
	}

	return resp.JsonUpdated(c, "Quiz updated", dto.ToLectureSessionsQuizDTO(quiz))
}

// =============================
// ❌ Delete Quiz By ID (soft delete)
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
