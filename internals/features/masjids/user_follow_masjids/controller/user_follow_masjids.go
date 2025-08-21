package controller

import (
	"math"
	"strconv"
	"time"

	masjidModel "masjidku_backend/internals/features/masjids/masjids/model"
	"masjidku_backend/internals/features/masjids/user_follow_masjids/model"
	helper "masjidku_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserFollowMasjidController struct {
	DB *gorm.DB
}

func NewUserFollowMasjidController(db *gorm.DB) *UserFollowMasjidController {
	return &UserFollowMasjidController{DB: db}
}

// =====================================================
// ✅ Follow masjid (idempotent)
// Body: { "masjid_id": "<uuid>" }
// =====================================================
func (ctrl *UserFollowMasjidController) FollowMasjid(c *fiber.Ctx) error {
	var input struct {
		MasjidID string `json:"masjid_id"`
	}
	if err := c.BodyParser(&input); err != nil || input.MasjidID == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Format input tidak valid / masjid_id kosong")
	}

	userIDStr, ok := c.Locals("user_id").(string)
	if !ok || userIDStr == "" {
		return helper.JsonError(c, fiber.StatusUnauthorized, "User tidak terautentikasi")
	}

	userUUID, err1 := uuid.Parse(userIDStr)
	masjidUUID, err2 := uuid.Parse(input.MasjidID)
	if err1 != nil || err2 != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "UUID user atau masjid tidak valid")
	}

	follow := model.UserFollowMasjidModel{
		FollowUserID:    userUUID,
		FollowMasjidID:  masjidUUID,
		FollowCreatedAt: time.Now(),
	}

	// Idempotent insert: jika sudah ada, DoNothing (tidak error)
	res := ctrl.DB.
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "follow_user_id"}, {Name: "follow_masjid_id"}},
			DoNothing: true,
		}).
		Create(&follow)

	if res.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal follow masjid")
	}

	if res.RowsAffected == 0 {
		// Sudah follow — balas OK dengan pesan informatif
		return helper.JsonOK(c, "Sudah mengikuti masjid ini", fiber.Map{
			"follow_user_id":   userUUID,
			"follow_masjid_id": masjidUUID,
		})
	}

	return helper.JsonCreated(c, "Berhasil follow masjid", follow)
}

// =====================================================
// 🚫 Unfollow masjid (idempotent)
// Body: { "masjid_id": "<uuid>" }
// =====================================================
func (ctrl *UserFollowMasjidController) UnfollowMasjid(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok || userIDStr == "" {
		return helper.JsonError(c, fiber.StatusUnauthorized, "User tidak terautentikasi")
	}

	var input struct {
		MasjidID string `json:"masjid_id"`
	}
	if err := c.BodyParser(&input); err != nil || input.MasjidID == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Masjid ID harus dikirim dalam body")
	}

	userUUID, err1 := uuid.Parse(userIDStr)
	masjidUUID, err2 := uuid.Parse(input.MasjidID)
	if err1 != nil || err2 != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "UUID user atau masjid tidak valid")
	}

	res := ctrl.DB.Delete(
		&model.UserFollowMasjidModel{},
		"follow_user_id = ? AND follow_masjid_id = ?",
		userUUID, masjidUUID,
	)
	if res.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal unfollow masjid")
	}
	if res.RowsAffected == 0 {
		// Tidak ada yang dihapus: anggap sudah tidak follow (idempotent)
		return helper.JsonOK(c, "Tidak mengikuti masjid ini", fiber.Map{
			"follow_user_id":   userUUID,
			"follow_masjid_id": masjidUUID,
			"unfollowed":       false,
		})
	}

	return helper.JsonDeleted(c, "Berhasil unfollow masjid", fiber.Map{
		"follow_user_id":   userUUID,
		"follow_masjid_id": masjidUUID,
	})
}

// =====================================================
// ❓ Cek status follow
// Query: ?masjid_id=<uuid>
// =====================================================
func (ctrl *UserFollowMasjidController) IsFollowing(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok || userIDStr == "" {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	masjidIDStr := c.Query("masjid_id")
	if masjidIDStr == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Parameter masjid_id wajib diisi")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "user_id tidak valid")
	}
	masjidID, err := uuid.Parse(masjidIDStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "masjid_id tidak valid")
	}

	var count int64
	if err := ctrl.DB.Model(&model.UserFollowMasjidModel{}).
		Where("follow_user_id = ? AND follow_masjid_id = ?", userID, masjidID).
		Count(&count).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengecek status follow")
	}

	return helper.JsonOK(c, "OK", fiber.Map{
		"is_following": count > 0,
	})
}

// =====================================================
// 📄 Daftar masjid yang diikuti (paginated)
// Query: ?page=1&limit=10
// NOTE: ganti nama tabel di Table("user_follow_masjid AS ufm")
// sesuai dengan nama sebenarnya di DB (singular/plural).
// =====================================================
func (ctrl *UserFollowMasjidController) GetFollowedMasjidsByUser(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok || userIDStr == "" {
		return helper.JsonError(c, fiber.StatusUnauthorized, "User tidak login")
	}
	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "User ID tidak valid")
	}

	page := parseIntDefault(c.Query("page"), 1)
	limit := parseIntDefault(c.Query("limit"), 10)
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	// Hitung total
	var total int64
	if err := ctrl.DB.
		Table("user_follow_masjid AS ufm"). // ⬅️ sesuaikan jika tabelmu bernama "user_follow_masjids"
		Where("ufm.follow_user_id = ?", userUUID).
		Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	type Result struct {
		masjidModel.MasjidModel
		FollowCreatedAt time.Time `json:"follow_created_at"`
	}

	var results []Result
	if err := ctrl.DB.
		Table("user_follow_masjid AS ufm").
		Select(`m.*, ufm.follow_created_at`).
		Joins("JOIN masjids m ON m.masjid_id = ufm.follow_masjid_id").
		Where("ufm.follow_user_id = ?", userUUID).
		Order("ufm.follow_created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(&results).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil daftar masjid yang diikuti")
	}

	pagination := fiber.Map{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": int(math.Ceil(float64(total) / float64(limit))),
	}

	return helper.JsonList(c, results, pagination)
}

// =============================
// utils
// =============================
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
