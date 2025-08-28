package controller

import (
	"masjidku_backend/internals/features/home/posts/dto"
	"masjidku_backend/internals/features/home/posts/model"
	helper "masjidku_backend/internals/helpers"
	"strconv"
	"strings"

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
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if err := validateTheme.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Ambil masjid_id dari token
	masjidID := c.Locals("masjid_id")
	if masjidID == nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Masjid ID not found in token")
	}

	theme := model.PostThemeModel{
		PostThemeName:        req.PostThemeName,
		PostThemeDescription: req.PostThemeDescription,
		PostThemeMasjidID:    masjidID.(string),
	}

	if err := ctrl.DB.Create(&theme).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to create theme")
	}

	return helper.JsonCreated(c, "Tema berhasil dibuat", dto.ToPostThemeDTO(theme))
}

// 🔄 Update Tema
func (ctrl *PostThemeController) UpdateTheme(c *fiber.Ctx) error {
	id := c.Params("id")

	var req dto.UpdatePostThemeRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if err := validateTheme.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var theme model.PostThemeModel
	if err := ctrl.DB.First(&theme, "post_theme_id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Theme not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to get theme")
	}

	theme.PostThemeName = req.PostThemeName
	theme.PostThemeDescription = req.PostThemeDescription

	if err := ctrl.DB.Save(&theme).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to update theme")
	}

	return helper.JsonOK(c, "Tema berhasil diperbarui", dto.ToPostThemeDTO(theme))
}

// 📄 Get Semua Tema (pagination opsional: ?page=1&page_size=20)
func (ctrl *PostThemeController) GetAllThemes(c *fiber.Ctx) error {
	// pagination
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int64
	if err := ctrl.DB.Model(&model.PostThemeModel{}).Where("post_theme_deleted_at IS NULL").Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to count themes")
	}

	var themes []model.PostThemeModel
	if err := ctrl.DB.
		Where("post_theme_deleted_at IS NULL").
		Preload("Masjid").
		Order("post_theme_created_at DESC").
		Limit(pageSize).Offset(offset).
		Find(&themes).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to retrieve themes")
	}

	result := make([]dto.PostThemeDTO, 0, len(themes))
	for _, t := range themes {
		result = append(result, dto.ToPostThemeDTO(t))
	}

	pagination := fiber.Map{
		"page":       page,
		"page_size":  pageSize,
		"total_data": total,
		"total_pages": func() int64 {
			if total == 0 {
				return 1
			}
			return (total + int64(pageSize) - 1) / int64(pageSize)
		}(),
		"has_next":  int64(offset+pageSize) < total,
		"has_prev":  page > 1,
		"next_page": func() int {
			if int64(offset+pageSize) < total {
				return page + 1
			}
			return page
		}(),
		"prev_page": func() int {
			if page > 1 {
				return page - 1
			}
			return page
		}(),
	}

	return helper.JsonList(c, result, pagination)
}

// 🔍 Get Tema by ID
func (ctrl *PostThemeController) GetThemeByID(c *fiber.Ctx) error {
	id := c.Params("id")

	var theme model.PostThemeModel
	if err := ctrl.DB.Preload("Masjid").First(&theme, "post_theme_id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Theme not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to retrieve theme")
	}

	return helper.JsonOK(c, "OK", dto.ToPostThemeDTO(theme))
}
// 📄 Get Tema by Masjid (dari token, pagination opsional)
func (ctrl *PostThemeController) GetThemesByMasjid(c *fiber.Ctx) error {
	masjidIDRaw := c.Locals("masjid_id")
	if masjidIDRaw == nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	masjidID := masjidIDRaw.(string)

	// pagination
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int64
	if err := ctrl.DB.Model(&model.PostThemeModel{}).
		Where("post_theme_masjid_id = ? AND post_theme_deleted_at IS NULL", masjidID).
		Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung daftar tema")
	}

	var themes []model.PostThemeModel
	if err := ctrl.DB.
		Where("post_theme_masjid_id = ? AND post_theme_deleted_at IS NULL", masjidID).
		Order("post_theme_created_at DESC").
		Limit(pageSize).Offset(offset).
		Find(&themes).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil daftar tema")
	}

	result := make([]dto.PostThemeDTO, 0, len(themes))
	for _, t := range themes {
		result = append(result, dto.ToPostThemeDTO(t))
	}

	pagination := fiber.Map{
		"page":       page,
		"page_size":  pageSize,
		"total_data": total,
		"total_pages": func() int64 {
			if total == 0 {
				return 1
			}
			return (total + int64(pageSize) - 1) / int64(pageSize)
		}(),
		"has_next":  int64(offset+pageSize) < total,
		"has_prev":  page > 1,
		"next_page": func() int {
			if int64(offset+pageSize) < total {
				return page + 1
			}
			return page
		}(),
		"prev_page": func() int {
			if page > 1 {
				return page - 1
			}
			return page
		}(),
	}

	return helper.JsonList(c, result, pagination)
}



// 🗑️ Hapus Tema (soft by default; hard with ?hard=true)
func (ctrl *PostThemeController) DeleteTheme(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "id is required")
	}

	hard := strings.EqualFold(c.Query("hard"), "true") || c.Query("hard") == "1"

	tx := ctrl.DB.WithContext(c.Context())
	var db *gorm.DB
	if hard {
		db = tx.Unscoped().Delete(&model.PostThemeModel{}, "post_theme_id = ?", id)
	} else {
		db = tx.Delete(&model.PostThemeModel{}, "post_theme_id = ?", id)
	}

	if db.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to delete theme")
	}
	if db.RowsAffected == 0 {
		return helper.JsonError(c, fiber.StatusNotFound, "Theme not found")
	}

	return helper.JsonDeleted(c, "Tema berhasil dihapus", fiber.Map{
		"id":   id,
		"hard": hard,
	})
}
