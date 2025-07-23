package controller

import (
	"masjidku_backend/internals/features/masjids/lecture_sessions/materials/dto"
	"masjidku_backend/internals/features/masjids/lecture_sessions/materials/model"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func (ctrl *LectureSessionsAssetController) FilterLectureSessionsAssets(c *fiber.Ctx) error {
	lectureSessionID := c.Query("lecture_session_id")
	if lectureSessionID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Missing lecture_session_id")
	}

	fileTypeQuery := c.Query("file_type") // bisa 1 atau 2,3,4
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

	var assets []model.LectureSessionsAssetModel
	if err := ctrl.DB.
		Where("lecture_sessions_asset_lecture_session_id = ? AND lecture_sessions_asset_file_type IN ?", lectureSessionID, fileTypes).
		Find(&assets).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve assets")
	}

	var response []dto.LectureSessionsAssetDTO
	for _, a := range assets {
		response = append(response, dto.ToLectureSessionsAssetDTO(a))
	}

	return c.JSON(response)
}
