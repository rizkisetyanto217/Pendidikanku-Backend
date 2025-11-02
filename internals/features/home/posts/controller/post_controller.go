package controller

import (
	"net/url"
	"schoolku_backend/internals/features/home/posts/dto"
	"schoolku_backend/internals/features/home/posts/model"
	helper "schoolku_backend/internals/helpers"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

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
		return helper.JsonError(c, fiber.StatusUnauthorized, "User belum login")
	}
	userID := userIDRaw.(string)

	schoolIDRaw := c.Locals("school_id")
	if schoolIDRaw == nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "School ID tidak ditemukan di token")
	}
	schoolID := schoolIDRaw.(string)

	title := c.FormValue("post_title")
	content := c.FormValue("post_content")
	postType := c.FormValue("post_type")
	isPublished := c.FormValue("post_is_published") == "true"
	themeID := c.FormValue("post_theme_id")

	if title == "" || content == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Judul dan konten wajib diisi")
	}

	var imageURL *string
	if file, err := c.FormFile("post_image_url"); err == nil && file != nil {
		url, err := helper.UploadImageToSupabase("posts", file)
		if err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal upload gambar")
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
		PostSchoolID:    &schoolID,
		PostUserID:      &userID,
	}

	if err := ctrl.DB.Create(&post).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat post")
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

	return helper.JsonCreated(c, "Post berhasil dibuat", dto.ToPostDTOWithTheme(post, theme, likeCount))
}

// 🔄 Update Post
func (ctrl *PostController) UpdatePost(c *fiber.Ctx) error {
	id := c.Params("id")

	var post model.PostModel
	if err := ctrl.DB.First(&post, "post_id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Post tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data post")
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
		// hapus lama (jika ada)
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
		// upload baru
		newURL, err := helper.UploadImageToSupabase("posts", file)
		if err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal upload gambar baru")
		}
		post.PostImageURL = &newURL
	}

	// 💾 Simpan
	if err := ctrl.DB.Save(&post).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal update post")
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

	return helper.JsonOK(c, "Post berhasil diperbarui", dto.ToPostDTOWithTheme(post, theme, likeCount))
}

// 📄 Get Semua Post (pagination opsional: ?page=1&page_size=20)
func (ctrl *PostController) GetAllPosts(c *fiber.Ctx) error {
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
	if err := ctrl.DB.Model(&model.PostModel{}).Where("post_deleted_at IS NULL").Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung total post")
	}

	var posts []model.PostModel
	if err := ctrl.DB.
		Where("post_deleted_at IS NULL").
		Preload("School").Preload("User").
		Order("post_created_at DESC").
		Limit(pageSize).Offset(offset).
		Find(&posts).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to retrieve posts")
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

	result := make([]dto.PostDTO, 0, len(posts))
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

	pagination := fiber.Map{
		"page":       page,
		"page_size":  pageSize,
		"total_data": total,
		"total_pages": func() int64 {
			if total == 0 {
				return 1
			}
			// ceil
			return (total + int64(pageSize) - 1) / int64(pageSize)
		}(),
		"has_next": int64(offset+pageSize) < total,
		"has_prev": page > 1,
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

// 📄 Get Posts by School ID (From Token, pagination opsional)
func (ctrl *PostController) GetPostsBySchool(c *fiber.Ctx) error {
	schoolIDRaw := c.Locals("school_id")
	if schoolIDRaw == nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "School ID tidak ditemukan di token")
	}
	schoolID := schoolIDRaw.(string)

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
	if err := ctrl.DB.Model(&model.PostModel{}).
		Where("post_school_id = ? AND post_deleted_at IS NULL", schoolID).
		Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung daftar postingan")
	}

	var posts []model.PostModel
	if err := ctrl.DB.
		Where("post_school_id = ? AND post_deleted_at IS NULL", schoolID).
		Order("post_created_at DESC").
		Limit(pageSize).Offset(offset).
		Find(&posts).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil daftar postingan")
	}

	// 🔎 Ambil semua theme ID unik dari post
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

	// Map theme
	themeMap := make(map[string]model.PostThemeModel)
	for _, t := range themes {
		themeMap[t.PostThemeID] = t
	}

	// Build result
	result := make([]dto.PostDTO, 0, len(posts))
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
		"has_next": int64(offset+pageSize) < total,
		"has_prev": page > 1,
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

// 🗑️ Hapus Post
func (ctrl *PostController) DeletePost(c *fiber.Ctx) error {
	id := c.Params("id")

	// opsional: cek ada
	var exists int64
	if err := ctrl.DB.Model(&model.PostModel{}).
		Where("post_id = ?", id).
		Count(&exists).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus post")
	}
	if exists == 0 {
		return helper.JsonError(c, fiber.StatusNotFound, "Post tidak ditemukan")
	}

	if err := ctrl.DB.Delete(&model.PostModel{}, "post_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus post")
	}

	return helper.JsonDeleted(c, "Post berhasil dihapus", fiber.Map{"id": id})
}
