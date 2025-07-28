package controller

import (
	"masjidku_backend/internals/features/home/posts/dto"
	"masjidku_backend/internals/features/home/posts/model"

	"github.com/gofiber/fiber/v2"
)

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


// 🌐 Get Post by ID (Public)
func (ctrl *PostController) GetPostByID(c *fiber.Ctx) error {
	id := c.Params("id")

	var post model.PostModel
	if err := ctrl.DB.First(&post, "post_id = ? AND post_deleted_at IS NULL", id).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Post tidak ditemukan")
	}

	// Ambil tema jika ada
	var theme *model.PostThemeModel
	if post.PostThemeID != nil {
		var temp model.PostThemeModel
		if err := ctrl.DB.First(&temp, "post_theme_id = ?", *post.PostThemeID).Error; err == nil {
			theme = &temp
		}
	}

	return c.JSON(fiber.Map{
		"message": "Berhasil mengambil detail post",
		"data":    dto.ToPostDTOWithTheme(post, theme),
	})
}
