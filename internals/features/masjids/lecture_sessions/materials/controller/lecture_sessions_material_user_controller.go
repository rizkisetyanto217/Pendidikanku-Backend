package controller

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gofiber/fiber/v2"

	lectureSessionModel "masjidku_backend/internals/features/masjids/lecture_sessions/main/model"
	"masjidku_backend/internals/features/masjids/lecture_sessions/materials/dto"
	"masjidku_backend/internals/features/masjids/lecture_sessions/materials/model"
	lectureModel "masjidku_backend/internals/features/masjids/lectures/main/model"
	helper "masjidku_backend/internals/helpers"
)

// =============================
// 🔍 Get Material by Lecture Sessions
// =============================
func (ctl *LectureSessionsMaterialController) FindByLectureSessionFiltered(c *fiber.Ctx) error {
	lectureSessionID := strings.TrimSpace(c.Query("lecture_session_id"))
	lectureID := strings.TrimSpace(c.Query("lecture_id"))

	if lectureSessionID == "" && lectureID == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "lecture_session_id atau lecture_id harus diisi")
	}

	filterType := c.Query("type")
	fmt.Println("🎯 Filter:", map[string]string{
		"lecture_session_id": lectureSessionID,
		"lecture_id":         lectureID,
		"type":               filterType,
	})

	var materials []model.LectureSessionsMaterialModel
	query := ctl.DB.Model(&model.LectureSessionsMaterialModel{})

	// Filter berdasarkan salah satu
	if lectureSessionID != "" {
		query = query.Where("lecture_sessions_material_lecture_session_id = ?", lectureSessionID)
	} else {
		// join ke table session jika filter by lecture_id
		query = query.Joins("JOIN lecture_sessions ON lecture_sessions.lecture_session_id = lecture_sessions_materials.lecture_sessions_material_lecture_session_id").
			Where("lecture_sessions.lecture_session_lecture_id = ?", lectureID)
	}

	// Kolom yang di-select berdasarkan type
	switch filterType {
	case "summary":
		query = query.Select(`
			lecture_sessions_materials.lecture_sessions_material_id,
			lecture_sessions_materials.lecture_sessions_material_summary,
			lecture_sessions_materials.lecture_sessions_material_lecture_session_id,
			lecture_sessions_materials.lecture_sessions_material_masjid_id,
			lecture_sessions_materials.lecture_sessions_material_created_at`)
	case "transcript":
		query = query.Select(`
			lecture_sessions_materials.lecture_sessions_material_id,
			lecture_sessions_materials.lecture_sessions_material_transcript_full,
			lecture_sessions_materials.lecture_sessions_material_lecture_session_id,
			lecture_sessions_materials.lecture_sessions_material_masjid_id,
			lecture_sessions_materials.lecture_sessions_material_created_at`)
	}

	if err := query.Find(&materials).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// Mapping ke DTO
	result := make([]dto.LectureSessionsMaterialDTO, 0, len(materials))
	for _, m := range materials {
		result = append(result, dto.ToLectureSessionsMaterialDTO(m))
	}

	return helper.JsonOK(c, "success", result)
}

// 🔍 Get Material by Lecture Sessions (By Slug)
func (ctl *LectureSessionsMaterialController) FindByLectureSessionFilteredBySlug(c *fiber.Ctx) error {
	lectureSessionSlug := strings.TrimSpace(c.Query("lecture_session_slug"))
	lectureSlug := strings.TrimSpace(c.Query("lecture_slug"))

	if lectureSessionSlug == "" && lectureSlug == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "lecture_session_slug atau lecture_slug harus diisi")
	}

	filterType := c.Query("type")
	fmt.Println("🎯 Filter:", map[string]string{
		"lecture_session_slug": lectureSessionSlug,
		"lecture_slug":         lectureSlug,
		"type":                 filterType,
	})

	var materials []model.LectureSessionsMaterialModel
	query := ctl.DB.Model(&model.LectureSessionsMaterialModel{})

	// 🔁 Filter by slug
	if lectureSessionSlug != "" {
		// Ambil UUID dari slug
		var session lectureSessionModel.LectureSessionModel
		if err := ctl.DB.Where("lecture_session_slug = ?", lectureSessionSlug).First(&session).Error; err != nil {
			return helper.JsonError(c, fiber.StatusNotFound, "Sesi kajian tidak ditemukan")
		}
		query = query.Where("lecture_sessions_material_lecture_session_id = ?", session.LectureSessionID)

	} else if lectureSlug != "" {
		// Ambil UUID dari lecture slug → cari semua session ID-nya
		var lecture lectureModel.LectureModel
		if err := ctl.DB.Where("lecture_slug = ?", lectureSlug).First(&lecture).Error; err != nil {
			return helper.JsonError(c, fiber.StatusNotFound, "Lecture tidak ditemukan")
		}

		query = query.Joins("JOIN lecture_sessions ON lecture_sessions.lecture_session_id = lecture_sessions_materials.lecture_sessions_material_lecture_session_id").
			Where("lecture_sessions.lecture_session_lecture_id = ?", lecture.LectureID)
	}

	// 🎯 Select kolom berdasarkan tipe
	switch filterType {
	case "summary":
		query = query.Select(`
			lecture_sessions_materials.lecture_sessions_material_id,
			lecture_sessions_materials.lecture_sessions_material_summary,
			lecture_sessions_materials.lecture_sessions_material_lecture_session_id,
			lecture_sessions_materials.lecture_sessions_material_masjid_id,
			lecture_sessions_materials.lecture_sessions_material_created_at`)
	case "transcript":
		query = query.Select(`
			lecture_sessions_materials.lecture_sessions_material_id,
			lecture_sessions_materials.lecture_sessions_material_transcript_full,
			lecture_sessions_materials.lecture_sessions_material_lecture_session_id,
			lecture_sessions_materials.lecture_sessions_material_masjid_id,
			lecture_sessions_materials.lecture_sessions_material_created_at`)
	}

	if err := query.Find(&materials).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// 🔁 Mapping ke DTO
	result := make([]dto.LectureSessionsMaterialDTO, 0, len(materials))
	for _, m := range materials {
		result = append(result, dto.ToLectureSessionsMaterialDTO(m))
	}

	return helper.JsonOK(c, "success", result)
}

// =============================
// 📚 Get Grouped Materials by Lecture ID
// =============================
func (ctl *LectureSessionsMaterialController) FindGroupedMaterialsByLectureID(c *fiber.Ctx) error {
	lectureID := strings.TrimSpace(c.Query("lecture_id"))
	filterType := c.Query("type") // summary atau transcript

	if lectureID == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "lecture_id wajib diisi")
	}

	var materials []model.LectureSessionsMaterialModel
	query := ctl.DB.Model(&model.LectureSessionsMaterialModel{}).
		Joins("JOIN lecture_sessions ON lecture_sessions.lecture_session_id = lecture_sessions_materials.lecture_sessions_material_lecture_session_id").
		Preload("LectureSession").
		Where("lecture_sessions.lecture_session_lecture_id = ?", lectureID)

	// Select kolom berdasarkan type
	switch filterType {
	case "summary":
		query = query.Select(`
			lecture_sessions_materials.lecture_sessions_material_id,
			lecture_sessions_materials.lecture_sessions_material_summary,
			lecture_sessions_materials.lecture_sessions_material_lecture_session_id,
			lecture_sessions_materials.lecture_sessions_material_masjid_id,
			lecture_sessions_materials.lecture_sessions_material_created_at`)
	case "transcript":
		query = query.Select(`
			lecture_sessions_materials.lecture_sessions_material_id,
			lecture_sessions_materials.lecture_sessions_material_transcript_full,
			lecture_sessions_materials.lecture_sessions_material_lecture_session_id,
			lecture_sessions_materials.lecture_sessions_material_masjid_id,
			lecture_sessions_materials.lecture_sessions_material_created_at`)
	}

	if err := query.Find(&materials).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	grouped := groupMaterialsBySession(materials)
	// tetap kembalikan array kosong jika tidak ada data
	return helper.JsonOK(c, "success", grouped)
}

// 📚 Get Grouped Materials by Lecture Slug
func (ctl *LectureSessionsMaterialController) FindGroupedMaterialsByLectureSlug(c *fiber.Ctx) error {
	lectureSlug := strings.TrimSpace(c.Query("lecture_slug"))
	filterType := c.Query("type") // summary atau transcript

	if lectureSlug == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "lecture_slug wajib diisi")
	}

	// ✅ Ambil Lecture ID berdasarkan slug
	var lecture lectureModel.LectureModel
	if err := ctl.DB.Where("lecture_slug = ?", lectureSlug).First(&lecture).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Lecture tidak ditemukan")
	}

	// ✅ Ambil semua material berdasarkan lecture_id
	var materials []model.LectureSessionsMaterialModel
	query := ctl.DB.Model(&model.LectureSessionsMaterialModel{}).
		Joins("JOIN lecture_sessions ON lecture_sessions.lecture_session_id = lecture_sessions_materials.lecture_sessions_material_lecture_session_id").
		Preload("LectureSession").
		Where("lecture_sessions.lecture_session_lecture_id = ?", lecture.LectureID)

	// ✅ Select kolom berdasarkan type
	switch filterType {
	case "summary":
		query = query.Select(`
			lecture_sessions_materials.lecture_sessions_material_id,
			lecture_sessions_materials.lecture_sessions_material_summary,
			lecture_sessions_materials.lecture_sessions_material_lecture_session_id,
			lecture_sessions_materials.lecture_sessions_material_masjid_id,
			lecture_sessions_materials.lecture_sessions_material_created_at`)
	case "transcript":
		query = query.Select(`
			lecture_sessions_materials.lecture_sessions_material_id,
			lecture_sessions_materials.lecture_sessions_material_transcript_full,
			lecture_sessions_materials.lecture_sessions_material_lecture_session_id,
			lecture_sessions_materials.lecture_sessions_material_masjid_id,
			lecture_sessions_materials.lecture_sessions_material_created_at`)
	}

	if err := query.Find(&materials).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	grouped := groupMaterialsBySession(materials)
	return helper.JsonOK(c, "success", grouped)
}

type GroupedMaterials struct {
	LectureSessionID    string                           `json:"lecture_session_id"`
	LectureSessionTitle string                           `json:"lecture_session_title"`
	Materials           []dto.LectureSessionsMaterialDTO `json:"materials"`
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
		dtoItem := dto.ToLectureSessionsMaterialDTO(item)
		groupMap[key] = append(groupMap[key], dtoItem)
	}

	result := make([]GroupedMaterials, 0, len(groupMap))
	for key, list := range groupMap {
		result = append(result, GroupedMaterials{
			LectureSessionID:    key.SessionID,
			LectureSessionTitle: key.Title,
			Materials:           list,
		})
	}

	// urutkan konsisten (descending by session_id seperti sebelumnya)
	sort.Slice(result, func(i, j int) bool {
		return result[i].LectureSessionID > result[j].LectureSessionID
	})

	return result
}
