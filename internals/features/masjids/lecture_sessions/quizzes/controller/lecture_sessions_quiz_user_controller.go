package controller

import (
	"masjidku_backend/internals/features/masjids/lecture_sessions/quizzes/dto"
	"masjidku_backend/internals/features/masjids/lecture_sessions/quizzes/model"
	questionModel "masjidku_backend/internals/features/masjids/lecture_sessions/questions/model"

	"github.com/gofiber/fiber/v2"
)

// =============================
// 🌐 Get Quiz By Masjid Slug (Public, full GORM)
// =============================
func (ctrl *LectureSessionsQuizController) GetQuizzesBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Slug masjid diperlukan")
	}

	// Cari masjid berdasarkan slug
	var masjid struct {
		MasjidID string `gorm:"column:masjid_id"`
	}
	if err := ctrl.DB.
		Table("masjids").
		Select("masjid_id").
		Where("masjid_slug = ?", slug).
		First(&masjid).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Masjid tidak ditemukan")
	}

	// Ambil quiz berdasarkan masjid_id
	var quizzes []model.LectureSessionsQuizModel
	if err := ctrl.DB.
		Where("lecture_sessions_quiz_masjid_id = ?", masjid.MasjidID).
		Find(&quizzes).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil quiz")
	}

	// Konversi ke DTO
	var result []dto.LectureSessionsQuizDTO
	for _, quiz := range quizzes {
		result = append(result, dto.ToLectureSessionsQuizDTO(quiz))
	}

	return c.JSON(result)
}


// ✅ GET /api/a/lecture-sessions-quiz/by-session/:id
func (ctrl *LectureSessionsQuizController) GetByLectureSessionID(c *fiber.Ctx) error {
	lectureSessionID := c.Params("id")
	if lectureSessionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Lecture session ID tidak ditemukan di URL",
		})
	}

	// Ambil satu quiz berdasarkan lecture_session_id
	var quiz model.LectureSessionsQuizModel
	if err := ctrl.DB.
		Where("lecture_sessions_quiz_lecture_session_id = ?", lectureSessionID).
		First(&quiz).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Quiz untuk sesi kajian ini tidak ditemukan",
		})
	}

	// Ambil soal-soal terkait quiz
	var questions []questionModel.LectureSessionsQuestionModel
	if err := ctrl.DB.
		Where("lecture_sessions_question_quiz_id = ?", quiz.LectureSessionsQuizID).
		Order("lecture_sessions_question_created_at ASC").
		Find(&questions).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal mengambil soal-soal quiz",
		})
	}

	// Response
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Quiz dan soal berhasil ditemukan",
		"data": fiber.Map{
			"quiz":      quiz,
			"questions": questions,
		},
	})
}
