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
	// ✅ Cek user login dari token
	userIDRaw := c.Locals("user_id")
	if userIDRaw == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "User belum login")
	}
	userID := userIDRaw.(string)

	// ✅ Cek masjid ID dari token
	masjidIDRaw := c.Locals("masjid_id")
	if masjidIDRaw == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	masjidID := masjidIDRaw.(string)

	// ✅ Ambil form value dari multipart/form-data
	title := c.FormValue("post_title")
	content := c.FormValue("post_content")
	postType := c.FormValue("post_type")
	isPublished := c.FormValue("post_is_published") == "true"
	themeID := c.FormValue("post_theme_id")

	// 🔎 Validasi wajib
	if title == "" || content == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Judul dan konten wajib diisi")
	}

	// ✅ Upload gambar jika ada
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

	// 🔁 Ubah themeID ke pointer jika ada
	var themeIDPtr *string
	if themeID != "" {
		themeIDPtr = &themeID
	}

	// ✅ Buat model post
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

	// 🧾 Simpan ke database
	if err := ctrl.DB.Create(&post).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat post")
	}

	// 🎉 Return response
	return c.Status(fiber.StatusCreated).JSON(dto.ToPostDTO(post))
}


func (ctrl *PostController) UpdatePost(c *fiber.Ctx) error {
	id := c.Params("id")

	// 🔍 Cari post yang ada
	var post model.PostModel
	if err := ctrl.DB.First(&post, "post_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Post tidak ditemukan")
	}

	// 🔁 Update field jika dikirim
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

	// 🖼️ Handle gambar jika ada file baru
	if file, err := c.FormFile("post_image_url"); err == nil && file != nil {
		// 🔁 Hapus gambar lama dari Supabase jika ada
		if post.PostImageURL != nil {
			parsed, err := url.Parse(*post.PostImageURL)
			if err == nil {
				rawPath := parsed.Path
				prefix := "/storage/v1/object/public/"
				cleaned := strings.TrimPrefix(rawPath, prefix)

				unescaped, err := url.QueryUnescape(cleaned)
				if err == nil {
					parts := strings.SplitN(unescaped, "/", 2)
					if len(parts) == 2 {
						bucket := parts[0]
						objectPath := parts[1]
						_ = helper.DeleteFromSupabase(bucket, objectPath)
					}
				}
			}
		}

		// ⬆️ Upload gambar baru
		newURL, err := helper.UploadImageToSupabase("posts", file)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal upload gambar baru")
		}
		post.PostImageURL = &newURL
	}

	// 💾 Simpan perubahan
	if err := ctrl.DB.Save(&post).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal update post")
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


// 📄 Get Posts by Masjid Slug (tanpa preload)
func (ctrl *PostController) GetPostsByMasjidSlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Slug masjid wajib diisi")
	}

	// 🔍 Cari masjid berdasarkan slug
	var masjid struct {
		MasjidID string `gorm:"column:masjid_id"`
	}
	if err := ctrl.DB.
		Table("masjids").
		Select("masjid_id").
		Where("masjid_slug = ?", slug).
		First(&masjid).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Masjid dengan slug tersebut tidak ditemukan")
	}

	// 🔍 Ambil semua post
	var posts []model.PostModel
	if err := ctrl.DB.
		Where("post_masjid_id = ? AND post_deleted_at IS NULL", masjid.MasjidID).
		Order("post_created_at DESC").
		Find(&posts).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil postingan masjid")
	}

	// 🔄 Ambil semua theme ID unik
	themeIDs := make(map[string]struct{})
	for _, p := range posts {
		if p.PostThemeID != nil {
			themeIDs[*p.PostThemeID] = struct{}{}
		}
	}

	// 🔁 Query semua tema sekaligus
	var themes []model.PostThemeModel
	if len(themeIDs) > 0 {
		var ids []string
		for id := range themeIDs {
			ids = append(ids, id)
		}
		ctrl.DB.Where("post_theme_id IN ?", ids).Find(&themes)
	}

	// 🔁 Buat map theme biar lookup cepat
	themeMap := make(map[string]model.PostThemeModel)
	for _, t := range themes {
		themeMap[t.PostThemeID] = t
	}

	// 🧾 Ubah ke DTO
	var result []dto.PostDTO
	for _, post := range posts {
		var theme *model.PostThemeModel
		if post.PostThemeID != nil {
			if t, ok := themeMap[*post.PostThemeID]; ok {
				theme = &t
			}
		}
		result = append(result, dto.ToPostDTOWithTheme(post, theme))
	}

	return c.JSON(fiber.Map{
		"message": "Berhasil mengambil postingan masjid berdasarkan slug",
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
