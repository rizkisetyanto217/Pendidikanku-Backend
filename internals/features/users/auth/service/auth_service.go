package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"os"
	"strings"
	"time"

	googleAuthIDTokenVerifier "github.com/futurenda/google-auth-id-token-verifier"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"masjidku_backend/internals/configs"
	classModel "masjidku_backend/internals/features/lembaga/classes/main/model"
	"masjidku_backend/internals/features/masjids/masjid_admins_teachers/model"
	progressUserService "masjidku_backend/internals/features/progress/progress/service"
	authHelper "masjidku_backend/internals/features/users/auth/helper"
	authModel "masjidku_backend/internals/features/users/auth/model"
	authRepo "masjidku_backend/internals/features/users/auth/repository"
	userModel "masjidku_backend/internals/features/users/user/model"
	userProfileService "masjidku_backend/internals/features/users/user/service"
	helpers "masjidku_backend/internals/helpers"
)

/* ========================== REGISTER ========================== */

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

	// Hash password
	passwordHash, err := authHelper.HashPassword(input.Password)
	if err != nil {
		return helpers.Error(c, fiber.StatusInternalServerError, "Password hashing failed")
	}
	input.Password = passwordHash

	// Create user
	if err := authRepo.CreateUser(db, &input); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate key") ||
			strings.Contains(strings.ToLower(err.Error()), "unique") {
			return helpers.Error(c, fiber.StatusBadRequest, "Email already registered")
		}
		return helpers.Error(c, fiber.StatusInternalServerError, "Failed to create user")
	}

	// ✅ Inisialisasi entitas turunan TANPA menjatuhkan register bila tabel belum ada
	if db.Migrator().HasTable("user_progress") {
		_ = progressUserService.CreateInitialUserProgress(db, input.ID)
	}
	if db.Migrator().HasTable("users_profile") {
		userProfileService.CreateInitialUserProfile(db, input.ID)
	}

	return helpers.SuccessWithCode(c, fiber.StatusCreated, "Registration successful", nil)
}

/* ========================== LOGIN ========================== */

func Login(db *gorm.DB, c *fiber.Ctx) error {
	var input struct {
		Identifier string `json:"identifier"`
		Password   string `json:"password"`
	}
	if err := c.BodyParser(&input); err != nil {
		return helpers.Error(c, fiber.StatusBadRequest, "Invalid input format")
	}
	input.Identifier = strings.TrimSpace(input.Identifier)

	if err := authHelper.ValidateLoginInput(input.Identifier, input.Password); err != nil {
		return helpers.Error(c, fiber.StatusBadRequest, err.Error())
	}

	// 🔍 Minimal user (id, password, is_active)
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

	// 🔄 Full user
	userFull, err := authRepo.FindUserByID(db, userLight.ID)
	if err != nil {
		return helpers.Error(c, fiber.StatusInternalServerError, "Gagal mengambil data user")
	}

	// Ambil daftar masjid per peran (tahan 42P01 dengan HasTable)
	masjidAdminIDs, masjidTeacherIDs, masjidStudentIDs, masjidIDs, err := collectMasjidRoleIDs(db, userFull.ID)
	if err != nil {
		return helpers.Error(c, fiber.StatusInternalServerError, err.Error())
	}

	// 🎫 Issue tokens
	return issueTokensWithRoles(c, db, *userFull, masjidAdminIDs, masjidTeacherIDs, masjidStudentIDs, masjidIDs)
}

/* ========================== HELPER: Kumpulkan peran ========================== */

func collectMasjidRoleIDs(db *gorm.DB, userID uuid.UUID) (
	masjidAdminIDs []string,
	masjidTeacherIDs []string,
	masjidStudentIDs []string,
	masjidIDs []string,
	err error,
) {
	adminSet := map[string]struct{}{}
	teacherSet := map[string]struct{}{}
	studentSet := map[string]struct{}{}

	// 1) Admin/DKM → masjid_admins
	if db.Migrator().HasTable("masjid_admins") {
		var adminMasjids []model.MasjidAdminModel
		if e := db.Where("masjid_admins_user_id = ? AND masjid_admins_is_active = true", userID).
			Find(&adminMasjids).Error; e != nil {
			return nil, nil, nil, nil, e
		}
		for _, m := range adminMasjids {
			adminSet[m.MasjidAdminsMasjidID.String()] = struct{}{}
		}
	}

	// 2) Teacher → masjid_teachers
	if db.Migrator().HasTable("masjid_teachers") {
		var teacherRows []model.MasjidTeacher
		if e := db.Where("masjid_teachers_user_id = ?", userID).
			Find(&teacherRows).Error; e != nil {
			return nil, nil, nil, nil, e
		}
		for _, t := range teacherRows {
			// sesuaikan field masjid ID di model kamu
			teacherSet[t.MasjidTeachersMasjidID] = struct{}{}
		}
	}

	// 3) Student → enrolment aktif di user_classes
	if db.Migrator().HasTable("user_classes") {
		var rows []struct {
			MasjidID *uuid.UUID `gorm:"column:user_classes_masjid_id"`
		}
		if e := db.Model(&classModel.UserClassesModel{}).
			Where("user_classes_user_id = ? AND user_classes_status = ? AND user_classes_ended_at IS NULL",
				userID, classModel.UserClassStatusActive).
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

	// Build slices (plus union)
	unionSet := map[string]struct{}{}
	masjidAdminIDs = make([]string, 0, len(adminSet))
	for id := range adminSet {
		masjidAdminIDs = append(masjidAdminIDs, id)
		unionSet[id] = struct{}{}
	}
	masjidTeacherIDs = make([]string, 0, len(teacherSet))
	for id := range teacherSet {
		masjidTeacherIDs = append(masjidTeacherIDs, id)
		unionSet[id] = struct{}{}
	}
	masjidStudentIDs = make([]string, 0, len(studentSet))
	for id := range studentSet {
		masjidStudentIDs = append(masjidStudentIDs, id)
		unionSet[id] = struct{}{}
	}
	masjidIDs = make([]string, 0, len(unionSet))
	for id := range unionSet {
		masjidIDs = append(masjidIDs, id)
	}
	return
}

/* ========================== ISSUE TOKENS ========================== */

// helper: HMAC-SHA256 -> []byte
func computeRefreshHash(token, secret string) []byte {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(token))
	return m.Sum(nil)
}

// helper: bikin *string kalau ada nilai
func strptr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func issueTokensWithRoles(
	c *fiber.Ctx,
	db *gorm.DB,
	user userModel.UserModel,
	masjidAdminIDs []string,
	masjidTeacherIDs []string,
	masjidStudentIDs []string,
	masjidIDs []string,
) error {
	const (
		accessTTL  = 24 * time.Hour
		refreshTTL = 7 * 24 * time.Hour
	)

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return helpers.Error(c, fiber.StatusInternalServerError, "JWT_SECRET belum diset")
	}

	refreshSecret := configs.JWTRefreshSecret
	if refreshSecret == "" {
		refreshSecret = os.Getenv("JWT_REFRESH_SECRET")
	}
	if refreshSecret == "" {
		return helpers.Error(c, fiber.StatusInternalServerError, "JWT_REFRESH_SECRET belum diset")
	}

	now := time.Now().UTC()

	// ---------- ACCESS TOKEN ----------
	accessClaims := jwt.MapClaims{
		"typ":                "access",
		"sub":                user.ID.String(),
		"id":                 user.ID.String(),
		"user_name":          user.UserName,
		"role":               user.Role,
		"masjid_admin_ids":   masjidAdminIDs,
		"masjid_teacher_ids": masjidTeacherIDs,
		"masjid_student_ids": masjidStudentIDs,
		"masjid_ids":         masjidIDs,
		"iat":                now.Unix(),
		"exp":                now.Add(accessTTL).Unix(),
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).
		SignedString([]byte(jwtSecret))
	if err != nil {
		return helpers.Error(c, fiber.StatusInternalServerError, "Gagal membuat access token")
	}

	// ---------- REFRESH TOKEN ----------
	refreshClaims := jwt.MapClaims{
		"typ": "refresh",
		"sub": user.ID.String(),
		"id":  user.ID.String(),
		"iat": now.Unix(),
		"exp": now.Add(refreshTTL).Unix(),
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).
		SignedString([]byte(refreshSecret))
	if err != nil {
		return helpers.Error(c, fiber.StatusInternalServerError, "Gagal membuat refresh token")
	}

	// Hash-kan token sebelum simpan
	token := computeRefreshHash(refreshToken, refreshSecret)

	ua := c.Get("User-Agent")
	ip := c.IP()

	// Simpan/rotasi refresh token di DB (pakai Token, bukan Token)
	if err := authRepo.CreateRefreshToken(db, &authModel.RefreshToken{
		UserID:    user.ID,
		Token: token,                // << ini yang baru
		ExpiresAt: now.Add(refreshTTL),
		UserAgent: strptr(ua),
		IP:        strptr(ip),
	}); err != nil {
		return helpers.Error(c, fiber.StatusInternalServerError, "Gagal menyimpan refresh token")
	}

	// ---------- SET COOKIES ----------
	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "None",
		Path:     "/",
		Expires:  now.Add(accessTTL),
	})
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken, // kirim plaintext ke client
		HTTPOnly: true,
		Secure:   true,
		SameSite: "None",
		Path:     "/",
		Expires:  now.Add(refreshTTL),
	})

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code":    200,
		"status":  "success",
		"message": "Login berhasil",
		"data": fiber.Map{
			"access_token": accessToken,
			"user": fiber.Map{
				"id":                 user.ID,
				"user_name":          user.UserName,
				"email":              user.Email,
				"role":               user.Role,
				"masjid_admin_ids":   masjidAdminIDs,
				"masjid_teacher_ids": masjidTeacherIDs,
				"masjid_student_ids": masjidStudentIDs,
				"masjid_ids":         masjidIDs,
			},
		},
	})
}

/* ========================== LOGIN GOOGLE ========================== */

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
			Password:         generateDummyPassword(), // dummy password (hash)
			GoogleID:         &googleID,
			Role:             "user",
			SecurityQuestion: "Created by Google",
			SecurityAnswer:   "google_auth",
			CreatedAt:        time.Now().UTC(),
			UpdatedAt:        time.Now().UTC(),
			IsActive:         true,
		}
		if err := authRepo.CreateUser(db, &newUser); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate key") ||
				strings.Contains(strings.ToLower(err.Error()), "unique") {
				return helpers.Error(c, fiber.StatusBadRequest, "Email already registered")
			}
			return helpers.Error(c, fiber.StatusInternalServerError, "Failed to create Google user")
		}

		if db.Migrator().HasTable("user_progress") {
			_ = progressUserService.CreateInitialUserProgress(db, newUser.ID)
		}
		if db.Migrator().HasTable("users_profile") {
			userProfileService.CreateInitialUserProfile(db, newUser.ID)
		}

		user = &newUser
	}

	// 🔄 Ambil data user full (guard is_active)
	userFull, err := authRepo.FindUserByID(db, user.ID)
	if err != nil {
		return helpers.Error(c, fiber.StatusInternalServerError, "Gagal mengambil data user")
	}
	if !userFull.IsActive {
		return helpers.Error(c, fiber.StatusForbidden, "Akun Anda telah dinonaktifkan. Hubungi admin.")
	}

	// Ambil daftar masjid per peran
	masjidAdminIDs, masjidTeacherIDs, masjidStudentIDs, masjidIDs, err := collectMasjidRoleIDs(db, userFull.ID)
	if err != nil {
		return helpers.Error(c, fiber.StatusInternalServerError, err.Error())
	}

	// 🎫 Issue tokens
	return issueTokensWithRoles(c, db, *userFull, masjidAdminIDs, masjidTeacherIDs, masjidStudentIDs, masjidIDs)
}

/* ========================== LOGOUT ========================== */

func Logout(db *gorm.DB, c *fiber.Ctx) error {
	// ✅ Ambil access token dari cookie
	accessToken := c.Cookies("access_token")
	if accessToken == "" {
		return helpers.Error(c, fiber.StatusUnauthorized, "Unauthorized - No access token in cookie")
	}

	// 🔒 Masukkan access token ke blacklist (pastikan repo melakukan hash jika DB simpan hash)
	if err := authRepo.BlacklistToken(db, accessToken, 4*24*time.Hour); err != nil {
		return helpers.Error(c, fiber.StatusInternalServerError, "Failed to blacklist token")
	}

	// 🧹 Hapus refresh token dari database (jika ada)
	refreshToken := c.Cookies("refresh_token")
	if refreshToken != "" {
		_ = authRepo.DeleteRefreshToken(db, refreshToken)
	}

	// ❌ Hapus cookie (pastikan Path + MaxAge)
	expired := time.Now().UTC().Add(-time.Hour)
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    "",
		HTTPOnly: true,
		Secure:   true,
		SameSite: "None",
		Path:     "/",
		Expires:  expired,
		MaxAge:   -1,
	})
	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    "",
		HTTPOnly: true,
		Secure:   true,
		SameSite: "None",
		Path:     "/",
		Expires:  expired,
		MaxAge:   -1,
	})

	return helpers.Success(c, "Logout successful", nil)
}

/* ========================== UTIL ========================== */

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

	// Bandingkan case-insensitive dan trim spasi
	if !strings.EqualFold(strings.TrimSpace(input.Answer), strings.TrimSpace(user.SecurityAnswer)) {
		return helpers.Error(c, fiber.StatusBadRequest, "Incorrect security answer")
	}

	return helpers.Success(c, "Security answer correct", fiber.Map{
		"email": user.Email,
	})
}
