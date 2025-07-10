package controller

import (
	"masjidku_backend/internals/features/masjids/lecture_sessions/materials/model"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type LectureSessionsContentController struct {
	DB *gorm.DB
}

func NewLectureSessionsContentController(db *gorm.DB) *LectureSessionsContentController {
	return &LectureSessionsContentController{
		DB: db,
	}
}

// GET /lecture-sessions-content/by-lecture?lecture_id=...
func (ctrl *LectureSessionsContentController) GetContentByLectureID(c *fiber.Ctx) error {
	lectureID := c.Query("lecture_id")
	if lectureID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "lecture_id wajib diisi",
		})
	}

	// Ambil semua session_id dari lecture tersebut
	var sessionIDs []string
	if err := ctrl.DB.
		Table("lecture_sessions").
		Where("lecture_session_lecture_id = ?", lectureID).
		Pluck("lecture_session_id", &sessionIDs).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal mengambil sesi kajian",
			"error":   err.Error(),
		})
	}

	var materials []model.LectureSessionsMaterialModel
	var assets []model.LectureSessionsAssetModel

	if len(sessionIDs) > 0 {
		ctrl.DB.
			Where("lecture_sessions_material_lecture_session_id IN ?", sessionIDs).
			Find(&materials)

		ctrl.DB.
			Where("lecture_sessions_asset_lecture_session_id IN ?", sessionIDs).
			Find(&assets)
	}

	// Gabungkan semuanya jadi satu array of map
	var content []map[string]interface{}
	for _, m := range materials {
		content = append(content, map[string]interface{}{
			"type":                     "material",
			"material_id":              m.LectureSessionsMaterialID,
			"material_title":           m.LectureSessionsMaterialTitle,
			"material_summary":         m.LectureSessionsMaterialSummary,
			"material_transcript_full": m.LectureSessionsMaterialTranscriptFull,
			"session_id":               m.LectureSessionsMaterialLectureSessionID,
			"created_at":               m.LectureSessionsMaterialCreatedAt,
		})
	}

	for _, a := range assets {
		content = append(content, map[string]interface{}{
			"type":            "asset",
			"asset_id":        a.LectureSessionsAssetID,
			"asset_title":     a.LectureSessionsAssetTitle,
			"asset_file_url":  a.LectureSessionsAssetFileURL,
			"asset_file_type": a.LectureSessionsAssetFileType,
			"session_id":      a.LectureSessionsAssetLectureSessionID,
			"created_at":      a.LectureSessionsAssetCreatedAt,
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Berhasil mengambil seluruh konten kajian berdasarkan lecture_id",
		"data":    content,
	})
}
