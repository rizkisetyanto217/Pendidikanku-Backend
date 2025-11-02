package controller

import (
	"database/sql"
	"errors"
	"math"
	"strconv"

	"schoolku_backend/internals/features/home/posts/dto"
	"schoolku_backend/internals/features/home/posts/model"
	helper "schoolku_backend/internals/helpers"

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

// =====================================================
// 🔄 Toggle Like (atomic, idempotent, race-safe)
// - Insert jika belum ada -> liked = TRUE
// - Jika sudah ada -> flip NOT is_liked
// - Selalu mengembalikan row hasil akhir
// =====================================================
func (ctrl *PostLikeController) ToggleLike(c *fiber.Ctx) error {
	var req dto.ToggleLikeRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if err := validateLike.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Unauthorized user")
	}

	// Ambil school_id via slug
	slug := c.Params("slug")
	if slug == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Slug school tidak ditemukan")
	}

	var schoolID string
	if err := ctrl.DB.
		Table("schools").
		Select("school_id").
		Where("school_slug = ?", slug).
		Scan(&schoolID).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil school")
	}
	if schoolID == "" {
		return helper.JsonError(c, fiber.StatusNotFound, "School tidak ditemukan")
	}

	// Pastikan post ada dan milik school yang sama (opsional tapi disarankan)
	var postSchoolID sql.NullString
	if err := ctrl.DB.
		Table("posts").
		Select("post_school_id").
		Where("post_id = ?", req.PostID).
		Scan(&postSchoolID).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memeriksa post")
	}
	if !postSchoolID.Valid {
		return helper.JsonError(c, fiber.StatusNotFound, "Post tidak ditemukan")
	}
	if postSchoolID.String != schoolID {
		return helper.JsonError(c, fiber.StatusForbidden, "Post tidak termasuk ke school ini")
	}

	// Atomic toggle via ON CONFLICT (post_id, user_id)
	var row model.PostLikeModel
	raw := `
		INSERT INTO post_likes (
			post_like_id,
			post_like_is_liked,
			post_like_post_id,
			post_like_user_id,
			post_like_school_id
		)
		VALUES (gen_random_uuid(), TRUE, @post_id, @user_id, @school_id)
		ON CONFLICT (post_like_post_id, post_like_user_id)
		DO UPDATE SET
			post_like_is_liked = NOT post_likes.post_like_is_liked,
			post_like_updated_at = NOW()
		RETURNING
			post_like_id,
			post_like_is_liked,
			post_like_post_id,
			post_like_user_id,
			post_like_school_id,
			post_like_updated_at
	`
	if err := ctrl.DB.
		Raw(raw, sql.Named("post_id", req.PostID), sql.Named("user_id", userID), sql.Named("school_id", schoolID)).
		Scan(&row).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal toggle like")
	}

	// Response
	dtoRes := dto.ToPostLikeDTO(row)
	// Jika barusan create (state awal TRUE), balas 201.
	// Kita tidak punya flag pasti dari query; gunakan heuristik:
	// - Kalau hasil akhirnya TRUE dan sebelumnya tidak ada, tetap 201.
	// - Untuk sederhana: selalu 200 OK kecuali kamu ingin pecah 201 saat FirstOrCreate.
	return helper.JsonOK(c, "Berhasil toggle like", dtoRes)
}

// =====================================================
// ✅ GET semua like (is_liked = TRUE) by post_id + pagination
// Query params:
//   - page (default 1)
//   - limit (default 10, max 100)
//
// =====================================================
func (ctrl *PostLikeController) GetAllLikesByPost(c *fiber.Ctx) error {
	postID := c.Params("post_id")
	if postID == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "post_id tidak ditemukan")
	}

	// pagination
	page := parseIntDefault(c.Query("page"), 1)
	limit := parseIntDefault(c.Query("limit"), 10)
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	// total
	var total int64
	if err := ctrl.DB.
		Model(&model.PostLikeModel{}).
		Where("post_like_post_id = ? AND post_like_is_liked = TRUE", postID).
		Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data like")
	}

	// data
	var likes []model.PostLikeModel
	if err := ctrl.DB.
		Where("post_like_post_id = ? AND post_like_is_liked = TRUE", postID).
		Order("post_like_updated_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&likes).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data like")
	}

	// map DTO
	response := make([]dto.PostLikeDTO, 0, len(likes))
	for _, like := range likes {
		response = append(response, dto.ToPostLikeDTO(like))
	}

	// pagination payload
	pagination := fiber.Map{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": int(math.Ceil(float64(total) / float64(limit))),
	}

	return helper.JsonList(c, response, pagination)
}

// =====================================================
// (Opsional) Cek status like user terhadap post (cepat)
// GET /posts/:post_id/like/me
// =====================================================
func (ctrl *PostLikeController) GetMyLikeStatus(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Unauthorized user")
	}
	postID := c.Params("post_id")
	if postID == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "post_id tidak ditemukan")
	}

	var isLiked bool
	err := ctrl.DB.
		Table("post_likes").
		Select("post_like_is_liked").
		Where("post_like_post_id = ? AND post_like_user_id = ?", postID, userID).
		Limit(1).
		Scan(&isLiked).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memeriksa status like")
	}

	return helper.JsonOK(c, "Status like", fiber.Map{
		"is_liked": isLiked,
	})
}

// ===============================
// utils
// ===============================
func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}
