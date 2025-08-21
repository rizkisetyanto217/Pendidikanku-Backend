package controller

import (
	"masjidku_backend/internals/constants"
	"masjidku_backend/internals/features/home/posts/dto"
	"masjidku_backend/internals/features/home/posts/model"
	helper "masjidku_backend/internals/helpers"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// 📄 Get Posts by Masjid Slug (pagination opsional: ?page=1&page_size=20)
func (ctrl *PostController) GetPostsByMasjidSlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Slug masjid wajib diisi")
	}

	// 🔍 Ambil masjid_id dari slug
	var masjid struct {
		MasjidID string `gorm:"column:masjid_id"`
	}
	if err := ctrl.DB.
		Table("masjids").
		Select("masjid_id").
		Where("masjid_slug = ? AND masjid_deleted_at IS NULL", slug).
		First(&masjid).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Masjid dengan slug tersebut tidak ditemukan")
	}

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

	// total
	var total int64
	if err := ctrl.DB.Model(&model.PostModel{}).
		Where("post_masjid_id = ? AND post_deleted_at IS NULL", masjid.MasjidID).
		Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung postingan masjid")
	}

	// 🔍 Ambil post masjid (page)
	var posts []model.PostModel
	if err := ctrl.DB.
		Where("post_masjid_id = ? AND post_deleted_at IS NULL", masjid.MasjidID).
		Order("post_created_at DESC").
		Limit(pageSize).Offset(offset).
		Find(&posts).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil postingan masjid")
	}

	// 🧠 Kumpulkan postIDs & themeIDs dari hasil halaman
	postIDs := make([]string, 0, len(posts))
	themeIDs := make(map[string]struct{})
	for _, p := range posts {
		postIDs = append(postIDs, p.PostID)
		if p.PostThemeID != nil {
			themeIDs[*p.PostThemeID] = struct{}{}
		}
	}

	// 🔍 Query tema sekaligus
	themeMap := make(map[string]model.PostThemeModel)
	if len(themeIDs) > 0 {
		var ids []string
		for id := range themeIDs {
			ids = append(ids, id)
		}
		var themes []model.PostThemeModel
		if err := ctrl.DB.Where("post_theme_id IN ?", ids).Find(&themes).Error; err == nil {
			for _, t := range themes {
				themeMap[t.PostThemeID] = t
			}
		}
	}

	// 🔢 Hitung like per post (hanya jika ada post)
	likeMap := make(map[string]int64)
	if len(postIDs) > 0 {
		type LikeCountResult struct {
			PostLikePostID string
			TotalLikes     int64
		}
		var likeCounts []LikeCountResult
		if err := ctrl.DB.
			Table("post_likes").
			Select("post_like_post_id, COUNT(*) as total_likes").
			Where("post_like_post_id IN ? AND post_like_is_liked = true", postIDs).
			Group("post_like_post_id").
			Scan(&likeCounts).Error; err == nil {
			for _, l := range likeCounts {
				likeMap[l.PostLikePostID] = l.TotalLikes
			}
		}
	}

	// ✅ Ambil user_id dari helper untuk flag is_liked
	userUUID := helper.GetUserUUID(c)
	likedPostMap := make(map[string]bool)
	if userUUID.String() != constants.DummyUserID.String() && len(postIDs) > 0 {
		var likedPosts []model.PostLikeModel
		if err := ctrl.DB.
			Where("post_like_user_id = ? AND post_like_post_id IN ? AND post_like_is_liked = true", userUUID.String(), postIDs).
			Find(&likedPosts).Error; err == nil {
			for _, lp := range likedPosts {
				likedPostMap[lp.PostLikePostID] = true
			}
		}
	}

	// 🧾 Ubah ke DTO
	result := make([]dto.PostDTO, 0, len(posts))
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

	// pagination payload
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

// 🌐 Get Post by ID (Public)
func (ctrl *PostController) GetPostByID(c *fiber.Ctx) error {
	id := c.Params("id")

	var post model.PostModel
	if err := ctrl.DB.First(&post, "post_id = ? AND post_deleted_at IS NULL", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Post tidak ditemukan")
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
	if err := ctrl.DB.
		Table("post_likes").
		Where("post_like_post_id = ? AND post_like_is_liked = true", post.PostID).
		Count(&likeCount).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil jumlah like")
	}

	return helper.JsonOK(c, "Berhasil mengambil detail post", dto.ToPostDTOWithTheme(post, theme, likeCount))
}
