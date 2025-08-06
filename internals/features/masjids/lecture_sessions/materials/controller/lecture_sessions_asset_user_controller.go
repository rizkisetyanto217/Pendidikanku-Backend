package controller

import (
	lectureSessionModel "masjidku_backend/internals/features/masjids/lecture_sessions/main/model"
	"masjidku_backend/internals/features/masjids/lecture_sessions/materials/dto"
	"masjidku_backend/internals/features/masjids/lecture_sessions/materials/model"
	lectureModel "masjidku_backend/internals/features/masjids/lectures/main/model"

	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func (ctrl *LectureSessionsAssetController) FilterLectureLectureSessionsAssets(c *fiber.Ctx) error {
	lectureSessionID := c.Query("lecture_session_id")
	lectureID := c.Query("lecture_id")
	fileTypeQuery := c.Query("file_type")

	// Validasi file_type wajib
	if fileTypeQuery == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Missing file_type")
	}

	// Parse file_type
	fileTypes := []int{}
	for _, s := range strings.Split(fileTypeQuery, ",") {
		ft, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid file_type value")
		}
		fileTypes = append(fileTypes, ft)
	}

	// Validasi: setidaknya salah satu dari lectureSessionID atau lectureID wajib ada
	if lectureSessionID == "" && lectureID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Either lecture_session_id or lecture_id must be provided")
	}

	var sessionIDs []string

	// Jika lectureID disediakan → ambil semua session_id dari lecture tersebut
	if lectureID != "" {
		if err := ctrl.DB.
			Table("lecture_sessions").
			Where("lecture_session_lecture_id = ?", lectureID).
			Pluck("lecture_session_id", &sessionIDs).Error; err != nil || len(sessionIDs) == 0 {
			return fiber.NewError(fiber.StatusNotFound, "Sesi kajian tidak ditemukan untuk lecture ini")
		}
	}

	// Ambil data asset
	var assets []model.LectureSessionsAssetModel
	query := ctrl.DB.Model(&model.LectureSessionsAssetModel{})

	if lectureSessionID != "" {
		query = query.Where("lecture_sessions_asset_lecture_session_id = ?", lectureSessionID)
	} else {
		query = query.Where("lecture_sessions_asset_lecture_session_id IN ?", sessionIDs)
	}

	if err := query.
		Where("lecture_sessions_asset_file_type IN ?", fileTypes).
		Find(&assets).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve assets")
	}

	// Mapping ke DTO
	var response []dto.LectureSessionsAssetDTO
	for _, a := range assets {
		response = append(response, dto.ToLectureSessionsAssetDTO(a))
	}

	return c.JSON(response)
}




func (ctrl *LectureSessionsAssetController) FilterLectureSessionsAssetsBySlug(c *fiber.Ctx) error {
	lectureSlug := c.Query("lecture_slug")
	lectureSessionSlug := c.Query("lecture_session_slug")
	fileTypeQuery := c.Query("file_type")

	// ✅ file_type wajib
	if fileTypeQuery == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Missing file_type")
	}

	// ✅ Parse file_type
	fileTypes := []int{}
	for _, s := range strings.Split(fileTypeQuery, ",") {
		ft, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid file_type value")
		}
		fileTypes = append(fileTypes, ft)
	}

	// ✅ Validasi setidaknya salah satu slug ada
	if lectureSlug == "" && lectureSessionSlug == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Either lecture_slug or lecture_session_slug must be provided")
	}

	var sessionIDs []string

	// Jika lecture_slug → ambil semua session_id dari lecture terkait
	if lectureSlug != "" {
		var lecture lectureModel.LectureModel
		if err := ctrl.DB.Where("lecture_slug = ?", lectureSlug).First(&lecture).Error; err != nil {
			return fiber.NewError(fiber.StatusNotFound, "Lecture tidak ditemukan")
		}

		if err := ctrl.DB.
			Table("lecture_sessions").
			Where("lecture_session_lecture_id = ?", lecture.LectureID).
			Pluck("lecture_session_id", &sessionIDs).Error; err != nil || len(sessionIDs) == 0 {
			return fiber.NewError(fiber.StatusNotFound, "Sesi kajian tidak ditemukan untuk lecture ini")
		}
	}

	// Jika lecture_session_slug → ambil satu session_id
	if lectureSessionSlug != "" {
		var session lectureSessionModel.LectureSessionModel
		if err := ctrl.DB.Where("lecture_session_slug = ?", lectureSessionSlug).First(&session).Error; err != nil {
			return fiber.NewError(fiber.StatusNotFound, "Sesi kajian tidak ditemukan")
		}
		sessionIDs = []string{session.LectureSessionID.String()}
	}

	// ✅ Query data asset
	var assets []model.LectureSessionsAssetModel
	if err := ctrl.DB.
		Model(&model.LectureSessionsAssetModel{}).
		Where("lecture_sessions_asset_lecture_session_id IN ?", sessionIDs).
		Where("lecture_sessions_asset_file_type IN ?", fileTypes).
		Find(&assets).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data asset")
	}

	// ✅ Mapping ke DTO
	var response []dto.LectureSessionsAssetDTO
	for _, a := range assets {
		response = append(response, dto.ToLectureSessionsAssetDTO(a))
	}

	return c.JSON(response)
}




// Get By Lecture ID
func (ctl *LectureSessionsAssetController) FindGroupedByLectureID(c *fiber.Ctx) error {
	lectureID := c.Query("lecture_id")
	fileTypes := c.Query("file_type") // contoh: "1,2"

	if lectureID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"message": "lecture_id wajib diisi",
		})
	}

	var assets []model.LectureSessionsAssetModel
	query := ctl.DB.Model(&model.LectureSessionsAssetModel{}).
		Joins("JOIN lecture_sessions ON lecture_sessions.lecture_session_id = lecture_sessions_assets.lecture_sessions_asset_lecture_session_id").
		Preload("LectureSession"). // 👈 WAJIB agar field Title bisa diakses
		Where("lecture_sessions.lecture_session_lecture_id = ?", lectureID)

	if fileTypes != "" {
		types := strings.Split(fileTypes, ",")
		query = query.Where("lecture_sessions_assets.lecture_sessions_asset_file_type IN (?)", types)
	}

	if err := query.
		Order("lecture_sessions_assets.lecture_sessions_asset_lecture_session_id DESC").
		Find(&assets).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"message": "gagal mengambil data",
			"error":   err.Error(),
		})
	}

	if len(assets) == 0 {
		return c.JSON(fiber.Map{
			"message": "success",
			"data":    []GroupedAssets{}, // tetap bentuk array kosong
		})
	}

	grouped := groupByLectureSessionID(assets)

	return c.JSON(fiber.Map{
		"message": "success",
		"data":    grouped,
	})
}


// Get By Lecture Slug
func (ctl *LectureSessionsAssetController) FindGroupedByLectureSlug(c *fiber.Ctx) error {
	lectureSlug := c.Query("lecture_slug")
	fileTypes := c.Query("file_type") // contoh: "1,2"

	if lectureSlug == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"message": "lecture_slug wajib diisi",
		})
	}

	// ✅ Ambil Lecture ID dari slug
	var lecture lectureModel.LectureModel
	if err := ctl.DB.Where("lecture_slug = ?", lectureSlug).First(&lecture).Error; err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"message": "Lecture tidak ditemukan",
		})
	}

	// ✅ Query asset berdasarkan lecture_id (hasil dari slug)
	var assets []model.LectureSessionsAssetModel
	query := ctl.DB.Model(&model.LectureSessionsAssetModel{}).
		Joins("JOIN lecture_sessions ON lecture_sessions.lecture_session_id = lecture_sessions_assets.lecture_sessions_asset_lecture_session_id").
		Preload("LectureSession"). // 👈 Agar field title/nama bisa diakses
		Where("lecture_sessions.lecture_session_lecture_id = ?", lecture.LectureID)

	// ✅ Jika ada filter file type
	if fileTypes != "" {
		types := strings.Split(fileTypes, ",")
		query = query.Where("lecture_sessions_assets.lecture_sessions_asset_file_type IN (?)", types)
	}

	// ✅ Ambil data asset
	if err := query.
		Order("lecture_sessions_assets.lecture_sessions_asset_lecture_session_id DESC").
		Find(&assets).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"message": "gagal mengambil data",
			"error":   err.Error(),
		})
	}

	// ✅ Tetap response array kosong jika tidak ada data
	if len(assets) == 0 {
		return c.JSON(fiber.Map{
			"message": "success",
			"data":    []GroupedAssets{},
		})
	}

	// ✅ Grouping berdasarkan lecture_session_id
	grouped := groupByLectureSessionID(assets)

	return c.JSON(fiber.Map{
		"message": "success",
		"data":    grouped,
	})
}


// Struktur grouping
type GroupedAssets struct {
	LectureSessionID    string                             `json:"lecture_session_id"`
	LectureSessionTitle string                             `json:"lecture_session_title"`
	Assets              []dto.LectureSessionsAssetResponse `json:"assets"`
}


// Grouping function (versi benar)
func groupByLectureSessionID(data []model.LectureSessionsAssetModel) []GroupedAssets {
	type groupKey struct {
		SessionID string
		Title     string
	}

	groupMap := make(map[groupKey][]dto.LectureSessionsAssetResponse)

	for _, item := range data {
		key := groupKey{
			SessionID: item.LectureSessionsAssetLectureSessionID,
			Title:     item.LectureSession.LectureSessionTitle, // pastikan relasi dimuat
		}

		resp := dto.ToLectureSessionsAssetResponse(item)
		groupMap[key] = append(groupMap[key], resp)
	}

	var result []GroupedAssets
	for key, assets := range groupMap {
		result = append(result, GroupedAssets{
			LectureSessionID:    key.SessionID,
			LectureSessionTitle: key.Title,
			Assets:              assets,
		})
	}

	// Urutkan berdasarkan session title atau ID (opsional)
	sort.Slice(result, func(i, j int) bool {
		return result[i].LectureSessionID > result[j].LectureSessionID
	})

	return result
}


