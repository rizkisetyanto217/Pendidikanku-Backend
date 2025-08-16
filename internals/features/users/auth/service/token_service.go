package service

// ======== Tambah import model admin/teacher jika belum ========
import (
	"masjidku_backend/internals/configs"
	matModel "masjidku_backend/internals/features/masjids/masjid_admins_teachers/model"
	authRepo "masjidku_backend/internals/features/users/auth/repository"
	helpers "masjidku_backend/internals/helpers"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ========================== REFRESH TOKEN ==========================
func RefreshToken(db *gorm.DB, c *fiber.Ctx) error {
	// 1) Ambil refresh token dari cookie atau body
	refreshToken := c.Cookies("refresh_token")
	if refreshToken == "" {
		var payload struct{ RefreshToken string `json:"refresh_token"` }
		if err := c.BodyParser(&payload); err != nil || payload.RefreshToken == "" {
			return helpers.Error(c, fiber.StatusUnauthorized, "No refresh token provided")
		}
		refreshToken = payload.RefreshToken
	}

	// 2) Pastikan token ada di DB (valid & belum dicabut)
	rt, err := authRepo.FindRefreshToken(db, refreshToken)
	if err != nil {
		return helpers.Error(c, fiber.StatusUnauthorized, "Invalid or expired refresh token")
	}

	// 3) Parse + cek expiry
	claims := jwt.MapClaims{}
	parser := jwt.Parser{SkipClaimsValidation: true}
	if _, err := parser.ParseWithClaims(refreshToken, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(configs.JWTRefreshSecret), nil
	}); err != nil {
		return helpers.Error(c, fiber.StatusUnauthorized, "Malformed refresh token")
	}
	exp, _ := claims["exp"].(float64)
	if time.Now().Unix() >= int64(exp) {
		return helpers.Error(c, fiber.StatusUnauthorized, "Refresh token expired")
	}

	// 4) Pastikan user masih aktif
	user, err := authRepo.FindUserByID(db, rt.UserID)
	if err != nil {
		return helpers.Error(c, fiber.StatusUnauthorized, "User not found")
	}
	if !user.IsActive {
		return helpers.Error(c, fiber.StatusForbidden, "Akun Anda telah dinonaktifkan")
	}

	// 5) Kumpulkan ulang semua masjid IDs
	adminIDs, teacherIDs, unionIDs, err := collectMasjidIDs(db, user.ID)
	if err != nil {
		return helpers.Error(c, fiber.StatusInternalServerError, "Gagal mengambil data masjid user")
	}

	// 6) ROTATE: hapus refresh token lama agar tidak bisa dipakai ulang
	_ = authRepo.DeleteRefreshToken(db, refreshToken)

	// 7) Keluarkan pasangan token baru dengan bentuk response yang SAMA
	return issueTokensWithRoles(c, db, *user, adminIDs, teacherIDs, unionIDs)
}

// ========================== HELPER: kumpulkan masjid IDs ==========================
func collectMasjidIDs(db *gorm.DB, userID uuid.UUID) (adminIDs []string, teacherIDs []string, unionIDs []string, err error) {
	adminSet := map[string]struct{}{}
	teacherSet := map[string]struct{}{}

	// Admin/DKM
	var adminRows []matModel.MasjidAdminModel
	if err = db.
		Where("masjid_admins_user_id = ? AND masjid_admins_is_active = true", userID).
		Find(&adminRows).Error; err != nil {
		return
	}
	for _, r := range adminRows {
		adminSet[r.MasjidID.String()] = struct{}{}
	}

	// Teacher
	var teacherRows []matModel.MasjidTeacher
	if err = db.
		Where("masjid_teachers_user_id = ?", userID).
		Find(&teacherRows).Error; err != nil {
		return
	}
	for _, r := range teacherRows {
		teacherSet[r.MasjidTeachersMasjidID] = struct{}{}
	}

	// Build slices
	for id := range adminSet {
		adminIDs = append(adminIDs, id)
	}
	for id := range teacherSet {
		teacherIDs = append(teacherIDs, id)
	}

	// Union
	seen := map[string]struct{}{}
	for id := range adminSet {
		seen[id] = struct{}{}
	}
	for id := range teacherSet {
		seen[id] = struct{}{}
	}
	for id := range seen {
		unionIDs = append(unionIDs, id)
	}
	return
}
