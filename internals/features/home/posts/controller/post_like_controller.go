package controller

import (
	"masjidku_backend/internals/features/home/posts/dto"
	"masjidku_backend/internals/features/home/posts/model"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var validateLike = validator.New()

type PostLikeController struct {
	DB *gorm.DB
}

func NewPostLikeController(db *gorm.DB) *PostLikeController {
	return &PostLikeController{DB: db}
}

// 🔄 Toggle Like
func (ctrl *PostLikeController) ToggleLike(c *fiber.Ctx) error {
	var req dto.ToggleLikeRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}
	if err := validateLike.Struct(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized user")
	}

	// 🔍 Ambil slug dari URL dan cari masjid_id
	slug := c.Params("slug")
	if slug == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Slug masjid tidak ditemukan")
	}

	var masjidID string
	err := ctrl.DB.
		Table("masjids").
		Select("masjid_id").
		Where("masjid_slug = ?", slug).
		Scan(&masjidID).Error
	if err != nil || masjidID == "" {
		return fiber.NewError(fiber.StatusNotFound, "Masjid tidak ditemukan")
	}

	// 🔁 Cek apakah like sudah ada
	var existing model.PostLikeModel
	err = ctrl.DB.Where("post_like_post_id = ? AND post_like_user_id = ?", req.PostID, userID).
		First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		newLike := model.PostLikeModel{
			PostLikePostID:   req.PostID,
			PostLikeUserID:   userID,
			PostLikeMasjidID: masjidID, // ✅ dari slug
			PostLikeIsLiked:  true,
		}
		if err := ctrl.DB.Create(&newLike).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to like post")
		}
		return c.Status(fiber.StatusCreated).JSON(dto.ToPostLikeDTO(newLike))
	} else if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to check like status")
	}

	// 🔁 Toggle status like
	existing.PostLikeIsLiked = !existing.PostLikeIsLiked
	if err := ctrl.DB.Save(&existing).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to update like status")
	}

	return c.JSON(dto.ToPostLikeDTO(existing))
}


// ✅ GET semua like (yang is_liked = true) berdasarkan post_id
func (ctrl *PostLikeController) GetAllLikesByPost(c *fiber.Ctx) error {
	postID := c.Params("post_id")
	if postID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "post_id tidak ditemukan")
	}

	var likes []model.PostLikeModel
	err := ctrl.DB.
		Where("post_like_post_id = ? AND post_like_is_liked = ?", postID, true).
		Find(&likes).Error
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data like")
	}

	var response []dto.PostLikeDTO
	for _, like := range likes {
		response = append(response, dto.ToPostLikeDTO(like))
	}

	return c.JSON(response)
}
