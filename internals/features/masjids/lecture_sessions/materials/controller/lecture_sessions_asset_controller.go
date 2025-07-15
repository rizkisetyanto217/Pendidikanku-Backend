package controller

import (
	"masjidku_backend/internals/features/masjids/lecture_sessions/materials/dto"
	"masjidku_backend/internals/features/masjids/lecture_sessions/materials/model"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var validate = validator.New() // ✅ Buat instance validator

type LectureSessionsAssetController struct {
	DB *gorm.DB
}

func NewLectureSessionsAssetController(db *gorm.DB) *LectureSessionsAssetController {
	return &LectureSessionsAssetController{DB: db}
}

func (ctrl *LectureSessionsAssetController) CreateLectureSessionsAsset(c *fiber.Ctx) error {
	var body dto.CreateLectureSessionsAssetRequest

	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request format")
	}

	if err := validate.Struct(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// ✅ Ambil masjid_id dari token (yang di-set di middleware)
	masjidID, ok := c.Locals("masjid_id").(string)
	if !ok || masjidID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "Masjid ID not found in token")
	}

	// Buat asset
	asset := model.LectureSessionsAssetModel{
		LectureSessionsAssetTitle:            body.LectureSessionsAssetTitle,
		LectureSessionsAssetFileURL:          body.LectureSessionsAssetFileURL,
		LectureSessionsAssetFileType:         body.LectureSessionsAssetFileType,
		LectureSessionsAssetLectureSessionID: body.LectureSessionsAssetLectureSessionID,
		LectureSessionsAssetMasjidID:         masjidID, // Ambil dari token
	}

	if err := ctrl.DB.Create(&asset).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to create asset")
	}

	return c.Status(fiber.StatusCreated).JSON(dto.ToLectureSessionsAssetDTO(asset))
}


// =============================
// 📄 Get All Assets
// =============================
func (ctrl *LectureSessionsAssetController) GetAllLectureSessionsAssets(c *fiber.Ctx) error {
	var assets []model.LectureSessionsAssetModel

	if err := ctrl.DB.Find(&assets).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve assets")
	}

	var response []dto.LectureSessionsAssetDTO
	for _, a := range assets {
		response = append(response, dto.ToLectureSessionsAssetDTO(a))
	}

	return c.JSON(response)
}

func (ctrl *LectureSessionsAssetController) FilterLectureSessionsAssets(c *fiber.Ctx) error {
	lectureSessionID := c.Query("lecture_session_id")
	if lectureSessionID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Missing lecture_session_id")
	}

	fileTypeQuery := c.Query("file_type") // bisa 1 atau 2,3,4
	if fileTypeQuery == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Missing file_type")
	}

	masjidID, ok := c.Locals("masjid_id").(string)
	if !ok || masjidID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "Masjid ID not found")
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
		Where("lecture_sessions_asset_lecture_session_id = ? AND lecture_sessions_asset_masjid_id = ? AND lecture_sessions_asset_file_type IN ?", lectureSessionID, masjidID, fileTypes).
		Find(&assets).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve assets")
	}

	var response []dto.LectureSessionsAssetDTO
	for _, a := range assets {
		response = append(response, dto.ToLectureSessionsAssetDTO(a))
	}

	return c.JSON(response)
}




// =============================
// ❌ Delete Asset
// =============================
func (ctrl *LectureSessionsAssetController) DeleteLectureSessionsAsset(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := ctrl.DB.Delete(&model.LectureSessionsAssetModel{}, "lecture_sessions_asset_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete asset")
	}

	return c.SendStatus(fiber.StatusNoContent)
}