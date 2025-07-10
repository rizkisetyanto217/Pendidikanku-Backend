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

	// ✅ Ambil user ID dari token (di-set oleh middleware auth)
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized user")
	}

	// 🔍 Cek apakah like sudah ada
	var existing model.PostLikeModel
	err := ctrl.DB.Where("post_like_post_id = ? AND post_like_user_id = ?", req.PostID, userID).
		First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		// 👍 Like belum ada → buat baru
		newLike := model.PostLikeModel{
			PostLikePostID:  req.PostID,
			PostLikeUserID:  userID,
			PostLikeIsLiked: true,
		}
		if err := ctrl.DB.Create(&newLike).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to like post")
		}
		return c.Status(fiber.StatusCreated).JSON(dto.ToPostLikeDTO(newLike))
	} else if err != nil {
		// ❌ Error saat cek
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to check like status")
	}

	// 🔁 Toggle nilai is_liked
	existing.PostLikeIsLiked = !existing.PostLikeIsLiked
	if err := ctrl.DB.Save(&existing).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to update like status")
	}

	return c.JSON(dto.ToPostLikeDTO(existing))
}
