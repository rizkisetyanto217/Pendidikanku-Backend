package controller

import (
	"masjidku_backend/internals/features/masjids/lecture_sessions/materials/dto"
	"masjidku_backend/internals/features/masjids/lecture_sessions/materials/model"
	"net/http"
	"sort"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// =============================
// 📚 Get Grouped Materials by Lecture ID
// =============================
func (ctl *LectureSessionsMaterialController) FindGroupedMaterialsByLectureID(c *fiber.Ctx) error {
	lectureID := strings.TrimSpace(c.Query("lecture_id"))
	filterType := c.Query("type") // summary atau transcript

	if lectureID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"message": "lecture_id wajib diisi",
		})
	}

	var materials []model.LectureSessionsMaterialModel
	query := ctl.DB.Model(&model.LectureSessionsMaterialModel{}).
		Joins("JOIN lecture_sessions ON lecture_sessions.lecture_session_id = lecture_sessions_materials.lecture_sessions_material_lecture_session_id").
		Preload("LectureSession").
		Where("lecture_sessions.lecture_session_lecture_id = ?", lectureID)

	// Select kolom berdasarkan type
	switch filterType {
	case "summary":
		query = query.Select("lecture_sessions_materials.lecture_sessions_material_id, lecture_sessions_materials.lecture_sessions_material_title, lecture_sessions_materials.lecture_sessions_material_summary, lecture_sessions_materials.lecture_sessions_material_lecture_session_id, lecture_sessions_materials.lecture_sessions_material_masjid_id, lecture_sessions_materials.lecture_sessions_material_created_at")
	case "transcript":
		query = query.Select("lecture_sessions_materials.lecture_sessions_material_id, lecture_sessions_materials.lecture_sessions_material_title, lecture_sessions_materials.lecture_sessions_material_transcript_full, lecture_sessions_materials.lecture_sessions_material_lecture_session_id, lecture_sessions_materials.lecture_sessions_material_masjid_id, lecture_sessions_materials.lecture_sessions_material_created_at")
	}

	// Eksekusi query
	if err := query.Find(&materials).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"message": "gagal mengambil data",
			"error":   err.Error(),
		})
	}

	if len(materials) == 0 {
		return c.JSON(fiber.Map{
			"message": "success",
			"data":    []GroupedMaterials{}, // kosong tetap array
		})
	}

	grouped := groupMaterialsBySession(materials)

	return c.JSON(fiber.Map{
		"message": "success",
		"data":    grouped,
	})
}


type GroupedMaterials struct {
	LectureSessionID    string                              `json:"lecture_session_id"`
	LectureSessionTitle string                              `json:"lecture_session_title"`
	Materials           []dto.LectureSessionsMaterialDTO    `json:"materials"`
}


func groupMaterialsBySession(data []model.LectureSessionsMaterialModel) []GroupedMaterials {
	type groupKey struct {
		SessionID string
		Title     string
	}

	groupMap := make(map[groupKey][]dto.LectureSessionsMaterialDTO)

	for _, item := range data {
		key := groupKey{
			SessionID: item.LectureSessionsMaterialLectureSessionID,
			Title:     item.LectureSession.LectureSessionTitle,
		}
		dto := dto.ToLectureSessionsMaterialDTO(item)
		groupMap[key] = append(groupMap[key], dto)
	}

	var result []GroupedMaterials
	for key, list := range groupMap {
		result = append(result, GroupedMaterials{
			LectureSessionID:    key.SessionID,
			LectureSessionTitle: key.Title,
			Materials:           list,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].LectureSessionID > result[j].LectureSessionID
	})

	return result
}
