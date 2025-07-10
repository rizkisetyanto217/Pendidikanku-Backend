package controller

import (
	"masjidku_backend/internals/features/home/posts/dto"
	"masjidku_backend/internals/features/home/posts/model"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var validatePost = validator.New()

type PostController struct {
	DB *gorm.DB
}

func NewPostController(db *gorm.DB) *PostController {
	return &PostController{DB: db}
}

// ➕ Buat Post
func (ctrl *PostController) CreatePost(c *fiber.Ctx) error {
	var req dto.CreatePostRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}
	if err := validatePost.Struct(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// ✅ Ambil user ID dari token (misal disimpan di Locals)
	userID := c.Locals("user_id").(string)

	// ✅ Ubah ke model via DTO
	post := dto.ToPostModel(req, &userID)

	if err := ctrl.DB.Create(&post).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to create post")
	}

	return c.Status(fiber.StatusCreated).JSON(dto.ToPostDTO(post))
}

// 🔄 Update Post
// 🔄 Update Post
func (ctrl *PostController) UpdatePost(c *fiber.Ctx) error {
	id := c.Params("id")

	var req dto.UpdatePostRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}
	if err := validatePost.Struct(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	var post model.PostModel
	if err := ctrl.DB.First(&post, "post_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Post not found")
	}

	// ✅ Pakai fungsi DTO untuk update model
	dto.UpdatePostModel(&post, req)

	if err := ctrl.DB.Save(&post).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to update post")
	}

	return c.JSON(dto.ToPostDTO(post))
}

// 📄 Get Semua Post
func (ctrl *PostController) GetAllPosts(c *fiber.Ctx) error {
	var posts []model.PostModel
	if err := ctrl.DB.Preload("Masjid").Preload("User").Find(&posts).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve posts")
	}

	var result []dto.PostDTO
	for _, post := range posts {
		result = append(result, dto.ToPostDTO(post))
	}

	return c.JSON(result)
}

// 🔍 Get Post by ID
func (ctrl *PostController) GetPostByID(c *fiber.Ctx) error {
	id := c.Params("id")

	var post model.PostModel
	if err := ctrl.DB.Preload("Masjid").Preload("User").First(&post, "post_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Post not found")
	}

	return c.JSON(dto.ToPostDTO(post))
}

// 🗑️ Hapus Post
func (ctrl *PostController) DeletePost(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := ctrl.DB.Delete(&model.PostModel{}, "post_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete post")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// =============================
// 📄 Get Posts by Masjid ID
// =============================// =============================
// 📄 Get Posts by Masjid ID
// =============================
func (ctrl *PostController) GetPostsByMasjid(c *fiber.Ctx) error {
	type RequestBody struct {
		MasjidID string `json:"masjid_id" validate:"required,uuid"`
	}

	var req RequestBody
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	// ✅ Ganti validate → validatePost
	if err := validatePost.Struct(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	var posts []model.PostModel
	if err := ctrl.DB.
		Where("post_masjid_id = ? AND post_deleted_at IS NULL", req.MasjidID).
		Order("post_created_at DESC").
		Find(&posts).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve posts")
	}

	var result []dto.PostDTO
	for _, post := range posts {
		result = append(result, dto.ToPostDTO(post))
	}

	return c.JSON(fiber.Map{
		"message": "Berhasil mengambil daftar postingan masjid",
		"data":    result,
	})
}
