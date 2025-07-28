package controller

import (
	"masjidku_backend/internals/features/home/posts/dto"
	"masjidku_backend/internals/features/home/posts/model"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var validateTheme = validator.New()

type PostThemeController struct {
	DB *gorm.DB
}

func NewPostThemeController(db *gorm.DB) *PostThemeController {
	return &PostThemeController{DB: db}
}

// ➕ Buat Tema
func (ctrl *PostThemeController) CreateTheme(c *fiber.Ctx) error {
	var req dto.CreatePostThemeRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}
	if err := validateTheme.Struct(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Ambil masjid_id dari token
	masjidID := c.Locals("masjid_id")
	if masjidID == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Masjid ID not found in token")
	}

	theme := model.PostThemeModel{
		PostThemeName:        req.PostThemeName,
		PostThemeDescription: req.PostThemeDescription,
		PostThemeMasjidID:    masjidID.(string),
	}

	if err := ctrl.DB.Create(&theme).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to create theme")
	}

	return c.Status(fiber.StatusCreated).JSON(dto.ToPostThemeDTO(theme))
}


// 🔄 Update Tema
func (ctrl *PostThemeController) UpdateTheme(c *fiber.Ctx) error {
	id := c.Params("id")

	var req dto.UpdatePostThemeRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}
	if err := validateTheme.Struct(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	var theme model.PostThemeModel
	if err := ctrl.DB.First(&theme, "post_theme_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Theme not found")
	}

	theme.PostThemeName = req.PostThemeName
	theme.PostThemeDescription = req.PostThemeDescription

	if err := ctrl.DB.Save(&theme).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to update theme")
	}

	return c.JSON(dto.ToPostThemeDTO(theme))
}

// 📄 Get Semua Tema
func (ctrl *PostThemeController) GetAllThemes(c *fiber.Ctx) error {
	var themes []model.PostThemeModel
	if err := ctrl.DB.Preload("Masjid").Find(&themes).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve themes")
	}

	var result []dto.PostThemeDTO
	for _, t := range themes {
		result = append(result, dto.ToPostThemeDTO(t))
	}

	return c.JSON(result)
}

// 🔍 Get Tema by ID
func (ctrl *PostThemeController) GetThemeByID(c *fiber.Ctx) error {
	id := c.Params("id")

	var theme model.PostThemeModel
	if err := ctrl.DB.Preload("Masjid").First(&theme, "post_theme_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Theme not found")
	}

	return c.JSON(dto.ToPostThemeDTO(theme))
}

// 🗑️ Hapus Tema
func (ctrl *PostThemeController) DeleteTheme(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := ctrl.DB.Delete(&model.PostThemeModel{}, "post_theme_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete theme")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// 📄 Get Tema by Masjid
func (ctrl *PostThemeController) GetThemesByMasjid(c *fiber.Ctx) error {
	type RequestBody struct {
		MasjidID string `json:"masjid_id" validate:"required,uuid"`
	}

	var req RequestBody
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}
	if err := validateTheme.Struct(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	var themes []model.PostThemeModel
	if err := ctrl.DB.Where("post_theme_masjid_id = ?", req.MasjidID).Order("post_theme_created_at DESC").Find(&themes).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve themes")
	}

	var result []dto.PostThemeDTO
	for _, t := range themes {
		result = append(result, dto.ToPostThemeDTO(t))
	}

	return c.JSON(fiber.Map{
		"message": "Berhasil mengambil daftar tema",
		"data":    result,
	})
}
