package controller

import (
	"masjidku_backend/internals/features/masjids/lecture_sessions/materials/dto"
	"masjidku_backend/internals/features/masjids/lecture_sessions/materials/model"

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

	// ✅ Validasi manual
	if err := validate.Struct(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	asset := model.LectureSessionsAssetModel{
		LectureSessionsAssetTitle:            body.LectureSessionsAssetTitle,
		LectureSessionsAssetFileURL:          body.LectureSessionsAssetFileURL,
		LectureSessionsAssetFileType:         body.LectureSessionsAssetFileType,
		LectureSessionsAssetLectureSessionID: body.LectureSessionsAssetLectureSessionID,
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

// =============================
// 🔍 Get Asset by ID
// =============================
func (ctrl *LectureSessionsAssetController) GetLectureSessionsAssetByID(c *fiber.Ctx) error {
	id := c.Params("id")

	var asset model.LectureSessionsAssetModel
	if err := ctrl.DB.First(&asset, "lecture_sessions_asset_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Asset not found")
	}

	return c.JSON(dto.ToLectureSessionsAssetDTO(asset))
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