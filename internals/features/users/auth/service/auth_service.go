package service

import (
	"strings"
	"time"

	googleAuthIDTokenVerifier "github.com/futurenda/google-auth-id-token-verifier"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"masjidku_backend/internals/configs"
	"masjidku_backend/internals/features/masjids/masjid_admins_teachers/model"
	authHelper "masjidku_backend/internals/features/users/auth/helper"
	authRepo "masjidku_backend/internals/features/users/auth/repository"
	userModel "masjidku_backend/internals/features/users/user/model"
	helpers "masjidku_backend/internals/helpers"

	progressUserService "masjidku_backend/internals/features/progress/progress/service"
	userProfileService "masjidku_backend/internals/features/users/user/service"
)

// ========================== REGISTER ==========================
func Register(db *gorm.DB, c *fiber.Ctx) error {
	var input userModel.UserModel
	if err := c.BodyParser(&input); err != nil {
		return helpers.Error(c, fiber.StatusBadRequest, "Invalid request body")
	}

	// Force set role ke "user" untuk mencegah manipulasi
	input.Role = "user"

	if err := authHelper.ValidateRegisterInput(input.UserName, input.Email, input.Password, input.SecurityAnswer); err != nil {
		return helpers.Error(c, fiber.StatusBadRequest, err.Error())
	}

	if err := input.Validate(); err != nil {
		return helpers.Error(c, fiber.StatusBadRequest, err.Error())
	}

	passwordHash, err := authHelper.HashPassword(input.Password)
	if err != nil {
		return helpers.Error(c, fiber.StatusInternalServerError, "Password hashing failed")
	}
	input.Password = passwordHash

	if err := authRepo.CreateUser(db, &input); err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return helpers.Error(c, fiber.StatusBadRequest, "Email already registered")
		}
		return helpers.Error(c, fiber.StatusInternalServerError, "Failed to create user")
	}

	// ✅ Buat progress dan profile default
	_ = progressUserService.CreateInitialUserProgress(db, input.ID)
	userProfileService.CreateInitialUserProfile(db, input.ID)

	return helpers.SuccessWithCode(c, fiber.StatusCreated, "Registration successful", nil)
}

// ========================== LOGIN ==========================
func Login(db *gorm.DB, c *fiber.Ctx) error {
	var input struct {
		Identifier string `json:"identifier"`
		Password   string `json:"password"`
	}
	if err := c.BodyParser(&input); err != nil {
		return helpers.Error(c, fiber.StatusBadRequest, "Invalid input format")
	}

	if err := authHelper.ValidateLoginInput(input.Identifier, input.Password); err != nil {
		return helpers.Error(c, fiber.StatusBadRequest, err.Error())
	}

	// 🔍 Ambil data minimal (id, password, is_active)
	userLight, err := authRepo.FindUserByEmailOrUsernameLight(db, input.Identifier)
	if err != nil {
		return helpers.Error(c, fiber.StatusUnauthorized, "Identifier atau Password salah")
	}

	if !userLight.IsActive {
		return helpers.Error(c, fiber.StatusForbidden, "Akun Anda telah dinonaktifkan. Hubungi admin.")
	}

	if err := authHelper.CheckPasswordHash(userLight.Password, input.Password); err != nil {
		return helpers.Error(c, fiber.StatusUnauthorized, "Identifier atau Password salah")
	}

	// 🔄 Ambil data lengkap user
	userFull, err := authRepo.FindUserByID(db, userLight.ID)
	if err != nil {
		return helpers.Error(c, fiber.StatusInternalServerError, "Gagal mengambil data user")
	}

	// Ambil daftar masjid_id yang dimiliki user jika role DKM
	var masjidIDs []string
	if userFull.Role == "dkm" {
		var adminMasjids []model.MasjidAdminModel
		if err := db.
			Where("masjid_admins_user_id = ? AND masjid_admins_is_active = true", userFull.ID).
			Find(&adminMasjids).Error; err != nil {
			return helpers.Error(c, fiber.StatusInternalServerError, "Gagal mengambil data masjid admin")
		}

		for _, m := range adminMasjids {
			masjidIDs = append(masjidIDs, m.MasjidID.String())
		}
	}

	return issueTokens(c, db, *userFull, masjidIDs)
}

// ========================== LOGIN GOOGLE ==========================
func LoginGoogle(db *gorm.DB, c *fiber.Ctx) error {
	var input struct {
		IDToken string `json:"id_token"`
	}
	if err := c.BodyParser(&input); err != nil {
		return helpers.Error(c, fiber.StatusBadRequest, "Invalid request body")
	}

	// 🔍 Verifikasi token Google
	v := googleAuthIDTokenVerifier.Verifier{}
	if err := v.VerifyIDToken(input.IDToken, []string{configs.GoogleClientID}); err != nil {
		return helpers.Error(c, fiber.StatusUnauthorized, "Invalid Google ID Token")
	}

	// ✅ Decode informasi dari token
	claimSet, err := googleAuthIDTokenVerifier.Decode(input.IDToken)
	if err != nil {
		return helpers.Error(c, fiber.StatusInternalServerError, "Failed to decode ID Token")
	}

	email, name, googleID := claimSet.Email, claimSet.Name, claimSet.Sub

	// 🔍 Cek apakah user sudah terdaftar dengan google_id
	user, err := authRepo.FindUserByGoogleID(db, googleID)
	if err != nil {
		// ❌ User belum ada → buat baru
		newUser := userModel.UserModel{
			UserName:         name,
			Email:            email,
			Password:         generateDummyPassword(), // dummy password
			GoogleID:         &googleID,
			Role:             "user",
			SecurityQuestion: "Created by Google",
			SecurityAnswer:   "google_auth",
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}

		if err := authRepo.CreateUser(db, &newUser); err != nil {
			if strings.Contains(err.Error(), "duplicate key") {
				return helpers.Error(c, fiber.StatusBadRequest, "Email already registered")
			}
			return helpers.Error(c, fiber.StatusInternalServerError, "Failed to create Google user")
		}

		// ✅ Buat user_progress dan user_profile
		_ = progressUserService.CreateInitialUserProgress(db, newUser.ID)
		userProfileService.CreateInitialUserProfile(db, newUser.ID)

		user = &newUser
	}

	// 🎟️ Buat access + refresh token
	return issueTokens(c, db, *user, nil)
}


// ========================== LOGOUT ==========================
func Logout(db *gorm.DB, c *fiber.Ctx) error {
	// ✅ Ambil access token dari cookie
	accessToken := c.Cookies("access_token")
	if accessToken == "" {
		return helpers.Error(c, fiber.StatusUnauthorized, "Unauthorized - No access token in cookie")
	}

	// 🔒 Masukkan access token ke blacklist
	if err := authRepo.BlacklistToken(db, accessToken, 4*24*time.Hour); err != nil {
		return helpers.Error(c, fiber.StatusInternalServerError, "Failed to blacklist token")
	}

	// 🧹 Hapus refresh token dari database (jika ada)
	refreshToken := c.Cookies("refresh_token")
	if refreshToken != "" {
		_ = authRepo.DeleteRefreshToken(db, refreshToken)
	}

	// ❌ Hapus cookie refresh_token
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    "",
		HTTPOnly: true,
		Secure:   true,
		SameSite: "None", // ⚠️ pakai "None" jika login dari frontend di domain berbeda
		Expires:  time.Now().Add(-time.Hour),
	})

	// ❌ Hapus cookie access_token juga
	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    "",
		HTTPOnly: true,
		Secure:   true,
		SameSite: "None",
		Expires:  time.Now().Add(-time.Hour),
	})

	return helpers.Success(c, "Logout successful", nil)
}
