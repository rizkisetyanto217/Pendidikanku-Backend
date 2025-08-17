package service

// ======== Tambah import model admin/teacher jika belum ========
import (
	"masjidku_backend/internals/configs"
	classModel "masjidku_backend/internals/features/lembaga/classes/main/model"

	// "masjidku_backend/internals/features/masjids/masjid_admins_teachers/model"
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

	// 3) Verifikasi signature + cek expiry (+ optional cek typ)
	claims := jwt.MapClaims{}
	parser := jwt.Parser{SkipClaimsValidation: true}
	if _, err := parser.ParseWithClaims(refreshToken, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(configs.JWTRefreshSecret), nil
	}); err != nil {
		return helpers.Error(c, fiber.StatusUnauthorized, "Malformed refresh token")
	}
	if typ, _ := claims["typ"].(string); typ != "refresh" {
		return helpers.Error(c, fiber.StatusUnauthorized, "Invalid token type")
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

	// 5) Kumpulkan ulang semua masjid IDs (admin, teacher, student, union)
	adminIDs, teacherIDs, studentIDs, unionIDs, err := collectMasjidIDsFull(db, user.ID)
	if err != nil {
		return helpers.Error(c, fiber.StatusInternalServerError, "Gagal mengambil data masjid user")
	}

	// 6) ROTATE: hapus refresh token lama agar tidak bisa dipakai ulang
	_ = authRepo.DeleteRefreshToken(db, refreshToken)

	// 7) Keluarkan pasangan token baru dengan bentuk response yang SAMA
	return issueTokensWithRoles(c, db, *user, adminIDs, teacherIDs, studentIDs, unionIDs)
}

// ==========================================================
// Helper: ambil daftar masjid per peran + union untuk user
// ==========================================================
func collectMasjidIDsFull(db *gorm.DB, userID uuid.UUID) (
	adminIDs []string,
	teacherIDs []string,
	studentIDs []string,
	unionIDs []string,
	err error,
) {
	adminSet := map[string]struct{}{}
	teacherSet := map[string]struct{}{}
	studentSet := map[string]struct{}{}

	// 1) Admin/DKM → masjid_admins (aktif)
	{
		var rows []matModel.MasjidAdminModel // sesuaikan tipe
		if e := db.
			Where("masjid_admins_user_id = ? AND masjid_admins_is_active = true", userID).
			Find(&rows).Error; e != nil {
			err = e
			return
		}
		for _, r := range rows {
			adminSet[r.MasjidID.String()] = struct{}{}
		}
	}

	// 2) Teacher → masjid_teachers
	{
		var rows []matModel.MasjidTeacher // sesuaikan tipe
		if e := db.
			Where("masjid_teachers_user_id = ?", userID).
			Find(&rows).Error; e != nil {
			err = e
			return
		}
		for _, r := range rows {
			teacherSet[r.MasjidTeachersMasjidID] = struct{}{}
		}
	}

	// 3) Student → user_classes aktif (status=active, ended_at IS NULL)
	{
		var rows []struct {
			MasjidID *uuid.UUID `gorm:"column:user_classes_masjid_id"`
		}
		if e := db.
			Model(&classModel.UserClassesModel{}).
			Where("user_classes_user_id = ? AND user_classes_status = ? AND user_classes_ended_at IS NULL",
				userID, classModel.UserClassStatusActive).
			Select("user_classes_masjid_id").
			Find(&rows).Error; e != nil {
			err = e
			return
		}
		for _, r := range rows {
			if r.MasjidID != nil {
				studentSet[r.MasjidID.String()] = struct{}{}
			}
		}
	}

	// Build slices
	adminIDs = make([]string, 0, len(adminSet))
	for id := range adminSet { adminIDs = append(adminIDs, id) }

	teacherIDs = make([]string, 0, len(teacherSet))
	for id := range teacherSet { teacherIDs = append(teacherIDs, id) }

	studentIDs = make([]string, 0, len(studentSet))
	for id := range studentSet { studentIDs = append(studentIDs, id) }

	// Union
	unionSet := map[string]struct{}{}
	for id := range adminSet   { unionSet[id] = struct{}{} }
	for id := range teacherSet { unionSet[id] = struct{}{} }
	for id := range studentSet { unionSet[id] = struct{}{} }

	unionIDs = make([]string, 0, len(unionSet))
	for id := range unionSet { unionIDs = append(unionIDs, id) }

	return
}