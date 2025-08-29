// internals/features/users/auth/service/token_service.go
package service

import (
	"os"
	"time"

	"masjidku_backend/internals/configs"
	classModel "masjidku_backend/internals/features/school/classes/main/model"
	matModel "masjidku_backend/internals/features/lembaga/masjid_admins_teachers/model"
	authModel "masjidku_backend/internals/features/users/auth/model"
	authRepo "masjidku_backend/internals/features/users/auth/repository"
	helpers "masjidku_backend/internals/helpers"

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

	// 2) Ambil secret untuk verifikasi JWT & hashing DB
	refreshSecret := configs.JWTRefreshSecret
	if refreshSecret == "" {
		refreshSecret = os.Getenv("JWT_REFRESH_SECRET")
	}
	if refreshSecret == "" {
		return helpers.Error(c, fiber.StatusInternalServerError, "JWT_REFRESH_SECRET belum diset")
	}

	// 3) Cari token by HASH (harus aktif & belum expired)
	tokenHash := computeRefreshHash(refreshToken, refreshSecret)
	rt, err := FindRefreshTokenByHashActive(db.WithContext(c.Context()), tokenHash)
	if err != nil {
		return helpers.Error(c, fiber.StatusUnauthorized, "Invalid or expired refresh token")
	}
	if rt.RevokedAt != nil || time.Now().After(rt.ExpiresAt) {
		return helpers.Error(c, fiber.StatusUnauthorized, "Refresh token expired")
	}

	// 4) Verifikasi signature & claim
	claims := jwt.MapClaims{}
	parser := jwt.Parser{SkipClaimsValidation: true}
	if _, err := parser.ParseWithClaims(refreshToken, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(refreshSecret), nil
	}); err != nil {
		return helpers.Error(c, fiber.StatusUnauthorized, "Malformed refresh token")
	}
	if typ, _ := claims["typ"].(string); typ != "refresh" {
		return helpers.Error(c, fiber.StatusUnauthorized, "Invalid token type")
	}
	if exp, ok := claims["exp"].(float64); ok && time.Now().Unix() >= int64(exp) {
		return helpers.Error(c, fiber.StatusUnauthorized, "Refresh token expired")
	}

	// 5) Pastikan user masih aktif
	user, err := authRepo.FindUserByID(db.WithContext(c.Context()), rt.UserID)
	if err != nil {
		return helpers.Error(c, fiber.StatusUnauthorized, "User not found")
	}
	if !user.IsActive {
		return helpers.Error(c, fiber.StatusForbidden, "Akun Anda telah dinonaktifkan")
	}

	// 6) Kumpulkan ulang masjid IDs (admin/teacher/student/union)
	adminIDs, teacherIDs, studentIDs, unionIDs, err := collectMasjidIDsFull(db.WithContext(c.Context()), user.ID)
	if err != nil {
		return helpers.Error(c, fiber.StatusInternalServerError, "Gagal mengambil data masjid user")
	}

	// 7) ROTATE: revoke token lama
	if err := RevokeRefreshTokenByID(db.WithContext(c.Context()), rt.ID); err != nil {
		return helpers.Error(c, fiber.StatusInternalServerError, "Gagal mencabut refresh token lama")
	}

	// 8) Issue pasangan token baru
	return issueTokensWithRoles(c, db, *user, adminIDs, teacherIDs, studentIDs, unionIDs)
}

// ==========================================================
// Helper: ambil daftar masjid per peran + union untuk user
//   - Toleran jika tabel peran belum ada (dev friendly, tanpa pgconn).
// ==========================================================
// ==========================================================
// Helper: ambil daftar masjid per peran + union untuk user
//   - Toleran jika tabel peran belum ada (dev friendly, tanpa pgconn).
// ==========================================================
func collectMasjidIDsFull(db *gorm.DB, userID uuid.UUID) (
	adminIDs []string,
	teacherIDs []string,
	studentIDs []string,
	unionIDs []string,
	err error,
) {
	// Bisa skip role lookup sementara (setup awal)
	if os.Getenv("AUTH_SKIP_ROLE_LOOKUP") == "1" {
		return nil, nil, nil, nil, nil
	}

	adminSet := map[string]struct{}{}
	teacherSet := map[string]struct{}{}
	studentSet := map[string]struct{}{}

	mig := db.Migrator()

	// 1) Admin/DKM → masjid_admins (aktif) — hanya query jika tabel ada
	if mig.HasTable(&matModel.MasjidAdminModel{}) || mig.HasTable("masjid_admins") {
		var rows []matModel.MasjidAdminModel
		if e := db.
			Where("masjid_admin_user_id = ? AND masjid_admin_is_active = TRUE", userID).
			Find(&rows).Error; e != nil {
			return nil, nil, nil, nil, e
		}
		for _, r := range rows {
			adminSet[r.MasjidAdminMasjidID.String()] = struct{}{}
		}
	}

	// 2) Teacher → masjid_teachers — hanya query jika tabel ada (singular cols + soft delete)
	if mig.HasTable(&matModel.MasjidTeacherModel{}) || mig.HasTable("masjid_teachers") {
		var rows []matModel.MasjidTeacherModel
		if e := db.
			Where("masjid_teacher_user_id = ? AND masjid_teacher_deleted_at IS NULL", userID).
			Find(&rows).Error; e != nil {
			return nil, nil, nil, nil, e
		}
		for _, r := range rows {
			teacherSet[r.MasjidTeacherMasjidID.String()] = struct{}{}
		}
	}

	// 3) Student → user_classes (aktif) — hanya query jika tabel ada
	if mig.HasTable(&classModel.UserClassesModel{}) || mig.HasTable("user_classes") {
		var rows []struct {
			MasjidID *uuid.UUID `gorm:"column:user_classes_masjid_id"`
		}
		if e := db.
			Model(&classModel.UserClassesModel{}).
			Where(`
				user_classes_user_id = ?
				AND user_classes_status = ?
				AND user_classes_deleted_at IS NULL
			`, userID, classModel.UserClassStatusActive).
			Select("user_classes_masjid_id").
			Find(&rows).Error; e != nil {
			return nil, nil, nil, nil, e
		}
		for _, r := range rows {
			if r.MasjidID != nil {
				studentSet[r.MasjidID.String()] = struct{}{}
			}
		}
	}

	// Build slices
	for id := range adminSet   { adminIDs = append(adminIDs, id) }
	for id := range teacherSet { teacherIDs = append(teacherIDs, id) }
	for id := range studentSet { studentIDs = append(studentIDs, id) }

	// Union
	unionMap := map[string]struct{}{}
	for id := range adminSet   { unionMap[id] = struct{}{} }
	for id := range teacherSet { unionMap[id] = struct{}{} }
	for id := range studentSet { unionMap[id] = struct{}{} }
	for id := range unionMap   { unionIDs = append(unionIDs, id) }

	return
}



// ========================== Mini-repo (tanpa dependensi baru) ==========================

// Cari refresh token yang aktif (belum di-revoke, belum expired)
func FindRefreshTokenByHashActive(db *gorm.DB, hash []byte) (*authModel.RefreshTokenModel, error) {
	var rt authModel.RefreshTokenModel
	if err := db.
		Where("token = ? AND revoked_at IS NULL AND expires_at > NOW()", hash).
		Limit(1).
		Find(&rt).Error; err != nil {
		return nil, err
	}
	if rt.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	return &rt, nil
}

// Revoke by ID
func RevokeRefreshTokenByID(db *gorm.DB, id uuid.UUID) error {
	now := time.Now().UTC()
	res := db.Model(&authModel.RefreshTokenModel{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", now)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
