package service

import (
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"masjidku_backend/internals/configs"
	authHelper "masjidku_backend/internals/features/users/auth/helper"
	authModel "masjidku_backend/internals/features/users/auth/model"
	authRepo "masjidku_backend/internals/features/users/auth/repository"
	userModel "masjidku_backend/internals/features/users/user/model"
	helpers "masjidku_backend/internals/helpers"
)

// ========================== REFRESH TOKEN ==========================
func RefreshToken(db *gorm.DB, c *fiber.Ctx) error {
	// 1️⃣ Ambil refresh_token dari cookie (default)
	refreshToken := c.Cookies("refresh_token")

	// 2️⃣ Atau fallback ke body JSON jika tidak ada di cookie
	if refreshToken == "" {
		var payload struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := c.BodyParser(&payload); err != nil || payload.RefreshToken == "" {
			return helpers.Error(c, fiber.StatusUnauthorized, "No refresh token provided")
		}
		refreshToken = payload.RefreshToken
	}

	// 🔍 Cek token ada di database
	rt, err := authRepo.FindRefreshToken(db, refreshToken)
	if err != nil {
		return helpers.Error(c, fiber.StatusUnauthorized, "Invalid or expired refresh token")
	}

	// 🧠 Validasi isi refresh token secara manual
	claims := jwt.MapClaims{}
	parser := jwt.Parser{SkipClaimsValidation: true}
	_, err = parser.ParseWithClaims(refreshToken, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(configs.JWTRefreshSecret), nil
	})
	if err != nil {
		log.Println("[ERROR] Failed to parse refresh token:", err)
		return helpers.Error(c, fiber.StatusUnauthorized, "Malformed refresh token")
	}

	exp, ok := claims["exp"].(float64)
	if !ok {
		return helpers.Error(c, fiber.StatusUnauthorized, "Refresh token missing expiration")
	}
	if time.Now().After(time.Unix(int64(exp), 0)) {
		return helpers.Error(c, fiber.StatusUnauthorized, "Refresh token expired")
	}

	// 🧑‍💼 Cek status aktif user sebelum lanjut
	var userStatus struct {
		IsActive bool
	}
	if err := db.Table("users").Select("is_active").Where("id = ?", rt.UserID).First(&userStatus).Error; err != nil {
		return helpers.Error(c, fiber.StatusUnauthorized, "User not found")
	}
	if !userStatus.IsActive {
		return helpers.Error(c, fiber.StatusForbidden, "Akun Anda telah dinonaktifkan")
	}

	// Jika aktif, baru ambil user lengkap untuk issueTokens
	user, err := authRepo.FindUserByID(db, rt.UserID)
	if err != nil {
		return helpers.Error(c, fiber.StatusUnauthorized, "User not found")
	}

	// 🔁 Kembalikan access_token baru + refresh_token baru
	return issueTokens(c, db, *user)
}

// ========================== ISSUE TOKEN ==========================
func issueTokens(c *fiber.Ctx, db *gorm.DB, user userModel.UserModel) error {
	// Durasi token
	const (
		accessTokenDuration  = 3600 * time.Minute
		refreshTokenDuration = 7 * 24 * time.Hour
	)

	// 🔐 Generate Access Token
	accessToken, accessExp, err := generateToken(user, db, configs.JWTSecret, accessTokenDuration)
	if err != nil {
		return helpers.Error(c, fiber.StatusInternalServerError, "Gagal membuat access token")
	}

	// 🔐 Generate Refresh Token
	refreshToken, refreshExp, err := generateToken(user, db, configs.JWTRefreshSecret, refreshTokenDuration)
	if err != nil {
		return helpers.Error(c, fiber.StatusInternalServerError, "Gagal membuat refresh token")
	}

	// 💾 Simpan Refresh Token ke DB
	rt := authModel.RefreshToken{
		UserID:    user.ID,
		Token:     refreshToken,
		ExpiresAt: refreshExp,
	}
	if err := authRepo.CreateRefreshToken(db, &rt); err != nil {
		return helpers.Error(c, fiber.StatusInternalServerError, "Gagal menyimpan refresh token")
	}

	// 🍪 Simpan Refresh Token di cookie
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
		Expires:  refreshExp,
	})

	// ✅ Response ke client
	return helpers.Success(c, "Login berhasil", fiber.Map{
		"access_token":        accessToken,
		"refresh_token_debug": refreshToken,      // ⚠️ Hapus ini di production
		"access_exp_unix":     accessExp.Unix(),  // Opsional: monitoring waktu
		"refresh_exp_unix":    refreshExp.Unix(), // Opsional
		"user": fiber.Map{
			"id":        user.ID,
			"user_name": user.UserName,
			"email":     user.Email,
			"role":      user.Role,
		},
	})
}

// ========================== GENERATE TOKEN ==========================
func generateToken(user userModel.UserModel, db *gorm.DB, secretKey string, duration time.Duration) (string, time.Time, error) {
	expiration := time.Now().Add(duration)

	// Ambil semua masjid_id dari tabel masjid_admins
	var masjidIDs []uuid.UUID
	err := db.Table("masjid_admins").
		Where("masjid_admins_user_id = ? AND masjid_admins_is_active = true", user.ID).
		Pluck("masjid_admins_masjid_id", &masjidIDs).Error
	if err != nil {
		return "", time.Time{}, err
	}

	// Konversi ke []string
	var masjidIDStrs []string
	for _, id := range masjidIDs {
		masjidIDStrs = append(masjidIDStrs, id.String())
	}

	// Tambahkan ke claims
	claims := jwt.MapClaims{
		"id":               user.ID.String(),
		"user_name":        user.UserName,
		"role":             user.Role,
		"masjid_admin_ids": masjidIDStrs, // 🟢 ditambahkan di sini
		"exp":              expiration.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secretKey))
	return tokenString, expiration, err
}

func generateDummyPassword() string {
	hash, _ := authHelper.HashPassword("RandomDummyPassword123!")
	return hash
}

func CheckSecurityAnswer(db *gorm.DB, c *fiber.Ctx) error {
	var input struct {
		Email  string `json:"email"`
		Answer string `json:"security_answer"`
	}

	if err := c.BodyParser(&input); err != nil {
		return helpers.Error(c, fiber.StatusBadRequest, "Invalid request format")
	}

	if err := authHelper.ValidateSecurityAnswerInput(input.Email, input.Answer); err != nil {
		return helpers.Error(c, fiber.StatusBadRequest, err.Error())
	}

	user, err := authRepo.FindUserByEmail(db, input.Email)
	if err != nil {
		return helpers.Error(c, fiber.StatusNotFound, "User not found")
	}

	if strings.TrimSpace(input.Answer) != strings.TrimSpace(user.SecurityAnswer) {
		return helpers.Error(c, fiber.StatusBadRequest, "Incorrect security answer")
	}

	return helpers.Success(c, "Security answer correct", fiber.Map{
		"email": user.Email,
	})
}
