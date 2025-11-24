package controller

// import (
// 	"math"
// 	"strconv"
// 	"time"

// 	schoolModel "madinahsalam_backend/internals/features/lembaga/school_yayasans/schools/model"
// 	"madinahsalam_backend/internals/features/lembaga/school_yayasans/user_follow_schools/model"
// 	helper "madinahsalam_backend/internals/helpers"

// 	"github.com/gofiber/fiber/v2"
// 	"github.com/google/uuid"
// 	"gorm.io/gorm"
// 	"gorm.io/gorm/clause"
// )

// type UserFollowSchoolController struct {
// 	DB *gorm.DB
// }

// func NewUserFollowSchoolController(db *gorm.DB) *UserFollowSchoolController {
// 	return &UserFollowSchoolController{DB: db}
// }

// // =====================================================
// // ✅ Follow school (idempotent)
// // Body: { "school_id": "<uuid>" }
// // =====================================================
// func (ctrl *UserFollowSchoolController) FollowSchool(c *fiber.Ctx) error {
// 	var input struct {
// 		SchoolID string `json:"school_id"`
// 	}
// 	if err := c.BodyParser(&input); err != nil || input.SchoolID == "" {
// 		return helper.JsonError(c, fiber.StatusBadRequest, "Format input tidak valid / school_id kosong")
// 	}

// 	userIDStr, ok := c.Locals("user_id").(string)
// 	if !ok || userIDStr == "" {
// 		return helper.JsonError(c, fiber.StatusUnauthorized, "User tidak terautentikasi")
// 	}

// 	userUUID, err1 := uuid.Parse(userIDStr)
// 	schoolUUID, err2 := uuid.Parse(input.SchoolID)
// 	if err1 != nil || err2 != nil {
// 		return helper.JsonError(c, fiber.StatusBadRequest, "UUID user atau school tidak valid")
// 	}

// 	follow := model.UserFollowSchoolModel{
// 		UserFollowSchoolUserID:   userUUID,
// 		UserFollowSchoolSchoolID: schoolUUID,
// 		// created_at auto oleh tag gorm: autoCreateTime, tapi tak masalah set manual juga:
// 		UserFollowSchoolCreatedAt: time.Now(),
// 	}

// 	// Idempotent insert: jika sudah ada, DoNothing (tidak error)
// 	res := ctrl.DB.
// 		Clauses(clause.OnConflict{
// 			Columns: []clause.Column{
// 				{Name: "user_follow_school_user_id"},
// 				{Name: "user_follow_school_school_id"},
// 			},
// 			DoNothing: true,
// 		}).
// 		Create(&follow)

// 	if res.Error != nil {
// 		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal follow school")
// 	}

// 	if res.RowsAffected == 0 {
// 		// Sudah follow — balas OK dengan pesan informatif
// 		return helper.JsonOK(c, "Sudah mengikuti school ini", fiber.Map{
// 			"user_follow_school_user_id":   userUUID,
// 			"user_follow_school_school_id": schoolUUID,
// 		})
// 	}

// 	return helper.JsonCreated(c, "Berhasil follow school", follow)
// }

// // =====================================================
// // 🚫 Unfollow school (idempotent)
// // Body: { "school_id": "<uuid>" }
// // =====================================================
// func (ctrl *UserFollowSchoolController) UnfollowSchool(c *fiber.Ctx) error {
// 	userIDStr, ok := c.Locals("user_id").(string)
// 	if !ok || userIDStr == "" {
// 		return helper.JsonError(c, fiber.StatusUnauthorized, "User tidak terautentikasi")
// 	}

// 	var input struct {
// 		SchoolID string `json:"school_id"`
// 	}
// 	if err := c.BodyParser(&input); err != nil || input.SchoolID == "" {
// 		return helper.JsonError(c, fiber.StatusBadRequest, "School ID harus dikirim dalam body")
// 	}

// 	userUUID, err1 := uuid.Parse(userIDStr)
// 	schoolUUID, err2 := uuid.Parse(input.SchoolID)
// 	if err1 != nil || err2 != nil {
// 		return helper.JsonError(c, fiber.StatusBadRequest, "UUID user atau school tidak valid")
// 	}

// 	res := ctrl.DB.Delete(
// 		&model.UserFollowSchoolModel{},
// 		"user_follow_school_user_id = ? AND user_follow_school_school_id = ?",
// 		userUUID, schoolUUID,
// 	)
// 	if res.Error != nil {
// 		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal unfollow school")
// 	}
// 	if res.RowsAffected == 0 {
// 		// Tidak ada yang dihapus: anggap sudah tidak follow (idempotent)
// 		return helper.JsonOK(c, "Tidak mengikuti school ini", fiber.Map{
// 			"user_follow_school_user_id":   userUUID,
// 			"user_follow_school_school_id": schoolUUID,
// 			"unfollowed":                   false,
// 		})
// 	}

// 	return helper.JsonDeleted(c, "Berhasil unfollow school", fiber.Map{
// 		"user_follow_school_user_id":   userUUID,
// 		"user_follow_school_school_id": schoolUUID,
// 	})
// }

// // =====================================================
// // ❓ Cek status follow
// // Query: ?school_id=<uuid>
// // =====================================================
// func (ctrl *UserFollowSchoolController) IsFollowing(c *fiber.Ctx) error {
// 	userIDStr, ok := c.Locals("user_id").(string)
// 	if !ok || userIDStr == "" {
// 		return helper.JsonError(c, fiber.StatusUnauthorized, "Unauthorized")
// 	}

// 	schoolIDStr := c.Query("school_id")
// 	if schoolIDStr == "" {
// 		return helper.JsonError(c, fiber.StatusBadRequest, "Parameter school_id wajib diisi")
// 	}

// 	userID, err := uuid.Parse(userIDStr)
// 	if err != nil {
// 		return helper.JsonError(c, fiber.StatusBadRequest, "user_id tidak valid")
// 	}
// 	schoolID, err := uuid.Parse(schoolIDStr)
// 	if err != nil {
// 		return helper.JsonError(c, fiber.StatusBadRequest, "school_id tidak valid")
// 	}

// 	var count int64
// 	if err := ctrl.DB.Model(&model.UserFollowSchoolModel{}).
// 		Where("user_follow_school_user_id = ? AND user_follow_school_school_id = ?", userID, schoolID).
// 		Count(&count).Error; err != nil {
// 		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengecek status follow")
// 	}

// 	return helper.JsonOK(c, "OK", fiber.Map{
// 		"is_following": count > 0,
// 	})
// }

// // =====================================================
// // 📄 Daftar school yang diikuti (paginated)
// // Query: ?page=1&limit=10
// // =====================================================
// func (ctrl *UserFollowSchoolController) GetFollowedSchoolsByUser(c *fiber.Ctx) error {
// 	userIDStr, ok := c.Locals("user_id").(string)
// 	if !ok || userIDStr == "" {
// 		return helper.JsonError(c, fiber.StatusUnauthorized, "User tidak login")
// 	}
// 	userUUID, err := uuid.Parse(userIDStr)
// 	if err != nil {
// 		return helper.JsonError(c, fiber.StatusBadRequest, "User ID tidak valid")
// 	}

// 	page := parseIntDefault(c.Query("page"), 1)
// 	limit := parseIntDefault(c.Query("limit"), 10)
// 	if limit <= 0 {
// 		limit = 10
// 	}
// 	if limit > 100 {
// 		limit = 100
// 	}
// 	offset := (page - 1) * limit

// 	// Hitung total
// 	var total int64
// 	if err := ctrl.DB.
// 		Table("user_follow_school AS ufm").
// 		Where("ufm.user_follow_school_user_id = ?", userUUID).
// 		Count(&total).Error; err != nil {
// 		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung total data")
// 	}

// 	type Result struct {
// 		schoolModel.SchoolModel
// 		FollowCreatedAt time.Time `json:"follow_created_at"`
// 	}

// 	var results []Result
// 	if err := ctrl.DB.
// 		Table("user_follow_school AS ufm").
// 		Select(`m.*, ufm.user_follow_school_created_at AS follow_created_at`).
// 		Joins("JOIN schools m ON m.school_id = ufm.user_follow_school_school_id").
// 		Where("ufm.user_follow_school_user_id = ?", userUUID).
// 		Order("ufm.user_follow_school_created_at DESC").
// 		Limit(limit).
// 		Offset(offset).
// 		Scan(&results).Error; err != nil {
// 		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil daftar school yang diikuti")
// 	}

// 	pagination := fiber.Map{
// 		"page":        page,
// 		"limit":       limit,
// 		"total":       total,
// 		"total_pages": int(math.Ceil(float64(total) / float64(limit))),
// 	}

// 	return helper.JsonList(c, results, pagination)
// }

// // =============================
// // utils
// // =============================
// func parseIntDefault(s string, def int) int {
// 	if s == "" {
// 		return def
// 	}
// 	v, err := strconv.Atoi(s)
// 	if err != nil {
// 		return def
// 	}
// 	return v
// }
