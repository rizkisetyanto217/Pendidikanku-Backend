package controller

import (
	"masjidku_backend/internals/features/masjids/lecture_sessions/quiz/dto"
	"masjidku_backend/internals/features/masjids/lecture_sessions/quiz/model"

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
