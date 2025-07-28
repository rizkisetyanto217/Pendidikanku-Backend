package controller

import (
	"masjidku_backend/internals/constants"
	"masjidku_backend/internals/features/home/posts/dto"
	"masjidku_backend/internals/features/home/posts/model"
	helper "masjidku_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
)

// 📄 Get Posts by Masjid Slug
func (ctrl *PostController) GetPostsByMasjidSlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Slug masjid wajib diisi")
	}

	// 🔍 Ambil masjid_id dari slug
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

	// 🔍 Ambil semua post masjid
	var posts []model.PostModel
	if err := ctrl.DB.
		Where("post_masjid_id = ? AND post_deleted_at IS NULL", masjid.MasjidID).
		Order("post_created_at DESC").
		Find(&posts).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil postingan masjid")
	}

	// 🧠 Siapkan postIDs dan themeIDs
	postIDs := make([]string, 0)
	themeIDs := make(map[string]struct{})
	for _, p := range posts {
		postIDs = append(postIDs, p.PostID)
		if p.PostThemeID != nil {
			themeIDs[*p.PostThemeID] = struct{}{}
		}
	}

	// 🔍 Query tema sekaligus
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

	// 🔢 Hitung like per post
	type LikeCountResult struct {
		PostLikePostID string
		TotalLikes     int64
	}
	var likeCounts []LikeCountResult
	ctrl.DB.
		Table("post_likes").
		Select("post_like_post_id, COUNT(*) as total_likes").
		Where("post_like_post_id IN ? AND post_like_is_liked = true", postIDs).
		Group("post_like_post_id").
		Scan(&likeCounts)

	likeMap := make(map[string]int64)
	for _, l := range likeCounts {
		likeMap[l.PostLikePostID] = l.TotalLikes
	}

	// ✅ Ambil user_id dari helper
	userUUID := helper.GetUserUUID(c)

	// 🔍 Ambil post yang disukai user jika bukan guest
	likedPostMap := make(map[string]bool)
	if userUUID.String() != constants.DummyUserID.String() {
		var likedPosts []model.PostLikeModel
		ctrl.DB.
			Where("post_like_user_id = ? AND post_like_post_id IN ? AND post_like_is_liked = true", userUUID.String(), postIDs).
			Find(&likedPosts)
		for _, lp := range likedPosts {
			likedPostMap[lp.PostLikePostID] = true
		}
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
		likeCount := likeMap[post.PostID]
		isLiked := likedPostMap[post.PostID]
		result = append(result, dto.ToPostDTOFull(post, theme, likeCount, isLiked))
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

	var theme *model.PostThemeModel
	if post.PostThemeID != nil {
		var temp model.PostThemeModel
		if err := ctrl.DB.First(&temp, "post_theme_id = ?", *post.PostThemeID).Error; err == nil {
			theme = &temp
		}
	}

	// 🔢 Ambil jumlah like
	var likeCount int64
	ctrl.DB.
		Table("post_likes").
		Where("post_like_post_id = ? AND post_like_is_liked = true", post.PostID).
		Count(&likeCount)

	return c.JSON(fiber.Map{
		"message": "Berhasil mengambil detail post",
		"data":    dto.ToPostDTOWithTheme(post, theme, likeCount),
	})
}
