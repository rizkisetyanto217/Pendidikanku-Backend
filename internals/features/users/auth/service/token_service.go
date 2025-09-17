// internals/features/users/auth/service/token_service.go
package service

import (
	"os"
	"time"

	"masjidku_backend/internals/configs"
	authModel "masjidku_backend/internals/features/users/auth/model"
	authRepo "masjidku_backend/internals/features/users/auth/repository"
	helper "masjidku_backend/internals/helpers"

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
			return helper.JsonError(c, fiber.StatusUnauthorized, "No refresh token provided")
		}
		refreshToken = payload.RefreshToken
	}

	// 2) Ambil secret untuk verifikasi JWT & hashing DB
	refreshSecret := configs.JWTRefreshSecret
	if refreshSecret == "" {
		refreshSecret = os.Getenv("JWT_REFRESH_SECRET")
	}
	if refreshSecret == "" {
		return helper.JsonError(c, fiber.StatusInternalServerError, "JWT_REFRESH_SECRET belum diset")
	}

	// 3) Cari token by HASH (harus aktif & belum expired)
	tokenHash := computeRefreshHash(refreshToken, refreshSecret)
	rt, err := FindRefreshTokenByHashActive(db.WithContext(c.Context()), tokenHash)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Invalid or expired refresh token")
	}
	if rt.RevokedAt != nil || time.Now().After(rt.ExpiresAt) {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Refresh token expired")
	}

	// 4) Verifikasi signature & claim (skip claims time; cek manual)
	claims := jwt.MapClaims{}
	parser := jwt.Parser{SkipClaimsValidation: true}
	if _, err := parser.ParseWithClaims(refreshToken, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(refreshSecret), nil
	}); err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Malformed refresh token")
	}
	if typ, _ := claims["typ"].(string); typ != "refresh" {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Invalid token type")
	}
	if exp, ok := claims["exp"].(float64); ok && time.Now().Unix() >= int64(exp) {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Refresh token expired")
	}

	// 5) Pastikan user masih aktif
	user, err := authRepo.FindUserByID(db.WithContext(c.Context()), rt.UserID)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "User not found")
	}
	if !user.IsActive {
		return helper.JsonError(c, fiber.StatusForbidden, "Akun Anda telah dinonaktifkan")
	}

	// 6) Ambil roles_global & masjid_roles
	rolesClaim, err := getUserRolesClaim(c.Context(), db, user.ID)
	if err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil roles user")
	}

	// 7) ROTATE: revoke token lama (idempotent)
	if err := RevokeRefreshTokenByID(db.WithContext(c.Context()), rt.ID); err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mencabut refresh token lama")
	}

	// 8) Issue pasangan token baru + set cookies + response (berbasis rolesClaim)
	if err := issueTokensWithRoles(c, db, *user, rolesClaim); err != nil {
		// Mapping error ke JSON konsisten
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	// Catatan:
	// - Jika issueTokensWithRoles SUDAH menulis response JSON (dan set cookies), kita cukup return nil.
	// - Jika tidak, ubah issueTokensWithRoles agar mengembalikan payload {access_token, refresh_token, expires_in, roles_claim}
	//   lalu kirim dengan:
	//   return helper.JsonOK(c, "Token refreshed", payload)

	return nil
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