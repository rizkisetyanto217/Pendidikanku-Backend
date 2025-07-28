package controller

import (
	"masjidku_backend/internals/features/home/posts/dto"
	"masjidku_backend/internals/features/home/posts/model"
	helper "masjidku_backend/internals/helpers"
	"net/url"
	"strings"

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
	userIDRaw := c.Locals("user_id")
	if userIDRaw == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "User belum login")
	}
	userID := userIDRaw.(string)

	masjidIDRaw := c.Locals("masjid_id")
	if masjidIDRaw == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	masjidID := masjidIDRaw.(string)

	title := c.FormValue("post_title")
	content := c.FormValue("post_content")
	postType := c.FormValue("post_type")
	isPublished := c.FormValue("post_is_published") == "true"
	themeID := c.FormValue("post_theme_id")

	if title == "" || content == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Judul dan konten wajib diisi")
	}

	var imageURL *string
	if file, err := c.FormFile("post_image_url"); err == nil && file != nil {
		url, err := helper.UploadImageToSupabase("posts", file)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal upload gambar")
		}
		imageURL = &url
	} else if val := c.FormValue("post_image_url"); val != "" {
		imageURL = &val
	}

	var themeIDPtr *string
	if themeID != "" {
		themeIDPtr = &themeID
	}

	post := model.PostModel{
		PostTitle:       title,
		PostContent:     content,
		PostImageURL:    imageURL,
		PostIsPublished: isPublished,
		PostType:        postType,
		PostThemeID:     themeIDPtr,
		PostMasjidID:    &masjidID,
		PostUserID:      &userID,
	}

	if err := ctrl.DB.Create(&post).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat post")
	}

	// 🔎 Ambil theme jika ada
	var theme *model.PostThemeModel
	if post.PostThemeID != nil {
		var temp model.PostThemeModel
		if err := ctrl.DB.First(&temp, "post_theme_id = ?", *post.PostThemeID).Error; err == nil {
			theme = &temp
		}
	}

	// 🧮 LikeCount default 0 saat pertama dibuat
	likeCount := int64(0)

	return c.Status(fiber.StatusCreated).JSON(dto.ToPostDTOWithTheme(post, theme, likeCount))
}



func (ctrl *PostController) UpdatePost(c *fiber.Ctx) error {
	id := c.Params("id")

	var post model.PostModel
	if err := ctrl.DB.First(&post, "post_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Post tidak ditemukan")
	}

	if val := c.FormValue("post_title"); val != "" {
		post.PostTitle = val
	}
	if val := c.FormValue("post_content"); val != "" {
		post.PostContent = val
	}
	if val := c.FormValue("post_type"); val != "" {
		post.PostType = val
	}
	if val := c.FormValue("post_is_published"); val != "" {
		post.PostIsPublished = val == "true"
	}
	if val := c.FormValue("post_theme_id"); val != "" {
		post.PostThemeID = &val
	}

	// 🖼️ Handle gambar baru
	if file, err := c.FormFile("post_image_url"); err == nil && file != nil {
		if post.PostImageURL != nil {
			if parsed, err := url.Parse(*post.PostImageURL); err == nil {
				prefix := "/storage/v1/object/public/"
				cleaned := strings.TrimPrefix(parsed.Path, prefix)
				if unescaped, err := url.QueryUnescape(cleaned); err == nil {
					if parts := strings.SplitN(unescaped, "/", 2); len(parts) == 2 {
						_ = helper.DeleteFromSupabase(parts[0], parts[1])
					}
				}
			}
		}
		newURL, err := helper.UploadImageToSupabase("posts", file)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal upload gambar baru")
		}
		post.PostImageURL = &newURL
	}

	// 💾 Simpan
	if err := ctrl.DB.Save(&post).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal update post")
	}

	// 🔎 Ambil tema jika ada
	var theme *model.PostThemeModel
	if post.PostThemeID != nil {
		var temp model.PostThemeModel
		if err := ctrl.DB.First(&temp, "post_theme_id = ?", *post.PostThemeID).Error; err == nil {
			theme = &temp
		}
	}

	// 🔁 Hitung jumlah like
	var likeCount int64
	ctrl.DB.Model(&model.PostLikeModel{}).
		Where("post_like_post_id = ? AND post_like_is_liked = true", post.PostID).
		Count(&likeCount)

	return c.JSON(dto.ToPostDTOWithTheme(post, theme, likeCount))
}



// 📄 Get Semua Post
func (ctrl *PostController) GetAllPosts(c *fiber.Ctx) error {
	var posts []model.PostModel
	if err := ctrl.DB.Preload("Masjid").Preload("User").Find(&posts).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve posts")
	}

	// Ambil semua theme ID unik
	themeIDs := make(map[string]struct{})
	for _, post := range posts {
		if post.PostThemeID != nil {
			themeIDs[*post.PostThemeID] = struct{}{}
		}
	}
	var themes []model.PostThemeModel
	if len(themeIDs) > 0 {
		var ids []string
		for id := range themeIDs {
			ids = append(ids, id)
		}
		ctrl.DB.Where("post_theme_id IN ?", ids).Find(&themes)
	}
	themeMap := make(map[string]model.PostThemeModel)
	for _, t := range themes {
		themeMap[t.PostThemeID] = t
	}

	var result []dto.PostDTO
	for _, post := range posts {
		var theme *model.PostThemeModel
		if post.PostThemeID != nil {
			if t, ok := themeMap[*post.PostThemeID]; ok {
				theme = &t
			}
		}

		var likeCount int64
		ctrl.DB.Model(&model.PostLikeModel{}).
			Where("post_like_post_id = ? AND post_like_is_liked = true", post.PostID).
			Count(&likeCount)

		result = append(result, dto.ToPostDTOWithTheme(post, theme, likeCount))
	}

	return c.JSON(result)
}


// =============================
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

	// Ambil semua theme ID unik
	themeIDs := make(map[string]struct{})
	for _, post := range posts {
		if post.PostThemeID != nil {
			themeIDs[*post.PostThemeID] = struct{}{}
		}
	}
	var themes []model.PostThemeModel
	if len(themeIDs) > 0 {
		var ids []string
		for id := range themeIDs {
			ids = append(ids, id)
		}
		ctrl.DB.Where("post_theme_id IN ?", ids).Find(&themes)
	}
	themeMap := make(map[string]model.PostThemeModel)
	for _, t := range themes {
		themeMap[t.PostThemeID] = t
	}

	var result []dto.PostDTO
	for _, post := range posts {
		var theme *model.PostThemeModel
		if post.PostThemeID != nil {
			if t, ok := themeMap[*post.PostThemeID]; ok {
				theme = &t
			}
		}

		var likeCount int64
		ctrl.DB.Model(&model.PostLikeModel{}).
			Where("post_like_post_id = ? AND post_like_is_liked = true", post.PostID).
			Count(&likeCount)

		result = append(result, dto.ToPostDTOWithTheme(post, theme, likeCount))
	}

	return c.JSON(fiber.Map{
		"message": "Berhasil mengambil daftar postingan masjid",
		"data":    result,
	})
}



// 🗑️ Hapus Post
func (ctrl *PostController) DeletePost(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := ctrl.DB.Delete(&model.PostModel{}, "post_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete post")
	}

	return c.SendStatus(fiber.StatusNoContent)
}