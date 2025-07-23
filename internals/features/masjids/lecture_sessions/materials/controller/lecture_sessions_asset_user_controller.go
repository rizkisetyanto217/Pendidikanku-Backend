package controller

import (
	"masjidku_backend/internals/features/masjids/lecture_sessions/materials/dto"
	"masjidku_backend/internals/features/masjids/lecture_sessions/materials/model"
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
