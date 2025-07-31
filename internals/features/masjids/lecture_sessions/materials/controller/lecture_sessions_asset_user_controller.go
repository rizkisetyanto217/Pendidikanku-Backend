package controller

import (
	"masjidku_backend/internals/features/masjids/lecture_sessions/materials/dto"
	"masjidku_backend/internals/features/masjids/lecture_sessions/materials/model"
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

// Struktur grouping
type GroupedAssets struct {
	LectureSessionID string                             `json:"lecture_session_id"`
	Assets           []dto.LectureSessionsAssetResponse `json:"assets"`
}

// Grouping function (versi benar)
func groupByLectureSessionID(data []model.LectureSessionsAssetModel) []GroupedAssets {
	groupMap := make(map[string][]dto.LectureSessionsAssetResponse)

	for _, item := range data {
		sessionID := item.LectureSessionsAssetLectureSessionID
		resp := dto.ToLectureSessionsAssetResponse(item) // ✅ gunakan response bukan DTO
		groupMap[sessionID] = append(groupMap[sessionID], resp)
	}

	var result []GroupedAssets
	for sessionID, assets := range groupMap {
		result = append(result, GroupedAssets{
			LectureSessionID: sessionID,
			Assets:           assets,
		})
	}

	// Urutkan secara descending berdasarkan ID
	sort.Slice(result, func(i, j int) bool {
		return result[i].LectureSessionID > result[j].LectureSessionID
	})

	return result
}

