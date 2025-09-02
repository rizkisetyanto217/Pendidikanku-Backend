package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	googleAuthIDTokenVerifier "github.com/futurenda/google-auth-id-token-verifier"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"masjidku_backend/internals/configs"
	lembagaModel "masjidku_backend/internals/features/lembaga/masjid_admins_teachers/admins_teachers/model"
	progressUserService "masjidku_backend/internals/features/progress/progress/service"
	classModel "masjidku_backend/internals/features/school/classes/classes/model"
	authHelper "masjidku_backend/internals/features/users/auth/helper"
	authModel "masjidku_backend/internals/features/users/auth/model"
	authRepo "masjidku_backend/internals/features/users/auth/repository"
	userModel "masjidku_backend/internals/features/users/user/model"
	userProfileService "masjidku_backend/internals/features/users/user/service"
	helpers "masjidku_backend/internals/helpers"
)

/* ==========================
   Const & Types
========================== */

const (
	accessTTLDefault  = 24 * time.Hour
	refreshTTLDefault = 7 * 24 * time.Hour
)

type TeacherRecord struct {
	ID       uuid.UUID `json:"masjid_teacher_id"`
	MasjidID uuid.UUID `json:"masjid_id"`
}

/* ==========================
   Small Helpers
========================== */

func nowUTC() time.Time { return time.Now().UTC() }

func getJWTSecret() (string, error) {
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if secret == "" {
		return "", fiber.NewError(fiber.StatusInternalServerError, "JWT_SECRET belum diset")
	}
	return secret, nil
}

func getRefreshSecret() (string, error) {
	secret := strings.TrimSpace(configs.JWTRefreshSecret)
	if secret == "" {
		secret = strings.TrimSpace(os.Getenv("JWT_REFRESH_SECRET"))
	}
	if secret == "" {
		return "", fiber.NewError(fiber.StatusInternalServerError, "JWT_REFRESH_SECRET belum diset")
	}
	return secret, nil
}

func strptr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func computeRefreshHash(token, secret string) []byte {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(token))
	return m.Sum(nil)
}

func hasTable(db *gorm.DB, table string) bool {
	return db != nil && db.Migrator().HasTable(table)
}

/* ==========================
   REGISTER
========================== */

func Register(db *gorm.DB, c *fiber.Ctx) error {
	var input userModel.UserModel
	if err := c.BodyParser(&input); err != nil {
		return helpers.Error(c, fiber.StatusBadRequest, "Invalid request body")
	}

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
		low := strings.ToLower(err.Error())
		if strings.Contains(low, "duplicate key") || strings.Contains(low, "unique") {
			return helpers.Error(c, fiber.StatusBadRequest, "Email already registered")
		}
		return helpers.Error(c, fiber.StatusInternalServerError, "Failed to create user")
	}

	// Inisialisasi entitas turunan (best-effort)
	if hasTable(db, "user_progress") {
		_ = progressUserService.CreateInitialUserProgress(db, input.ID)
	}
	if hasTable(db, "users_profile") {
		userProfileService.CreateInitialUserProfile(db, input.ID)
	}

	return helpers.SuccessWithCode(c, fiber.StatusCreated, "Registration successful", nil)
}

/* ==========================
   LOGIN (username/email + password)
========================== */

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

	// Minimal user
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

	// Full user
	userFull, err := authRepo.FindUserByID(db, userLight.ID)
	if err != nil {
		return helpers.Error(c, fiber.StatusInternalServerError, "Gagal mengambil data user")
	}

	// Peran per masjid (tahan 42P01)
	masjidAdminIDs, masjidTeacherIDs, masjidStudentIDs, masjidIDs, err := collectMasjidRoleIDs(db, userFull.ID)
	if err != nil {
		return helpers.Error(c, fiber.StatusInternalServerError, err.Error())
	}

	// Issue tokens
	return issueTokensWithRoles(c, db, *userFull, masjidAdminIDs, masjidTeacherIDs, masjidStudentIDs, masjidIDs)
}

/* ==========================
   Helper: Kumpulkan peran per masjid
========================== */

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

	// Admin/DKM
	if hasTable(db, "masjid_admins") {
		var adminMasjids []lembagaModel.MasjidAdminModel
		if e := db.
			Where("masjid_admin_user_id = ? AND masjid_admin_is_active = TRUE", userID).
			Find(&adminMasjids).Error; e != nil {
			return nil, nil, nil, nil, e
		}
		for _, m := range adminMasjids {
			adminSet[m.MasjidAdminMasjidID.String()] = struct{}{}
		}
	}

	// Teacher
	if hasTable(db, "masjid_teachers") {
		var teacherRows []lembagaModel.MasjidTeacherModel
		if e := db.
			Where("masjid_teacher_user_id = ? AND masjid_teacher_deleted_at IS NULL", userID).
			Find(&teacherRows).Error; e != nil {
			return nil, nil, nil, nil, e
		}
		for _, t := range teacherRows {
			teacherSet[t.MasjidTeacherMasjidID.String()] = struct{}{}
		}
	}

	// Student (user_classes aktif)
	if hasTable(db, "user_classes") {
		var rows []struct {
			MasjidID *uuid.UUID `gorm:"column:user_classes_masjid_id"`
		}
		if e := db.Model(&classModel.UserClassesModel{}).
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

	// Build slices + union
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

/* ==========================
   Helper: Ambil Teacher Records (untuk role teacher)
========================== */

func fetchTeacherRecords(db *gorm.DB, user userModel.UserModel) []TeacherRecord {
	if !strings.EqualFold(strings.TrimSpace(user.Email), "teacher") {
		return nil
	}
	if !hasTable(db, "masjid_teachers") {
		return nil
	}

	var rows []lembagaModel.MasjidTeacherModel
	if err := db.
		Where("masjid_teacher_user_id = ? AND masjid_teacher_deleted_at IS NULL", user.ID).
		Find(&rows).Error; err != nil {
		log.Printf("[WARN] fetch teacher_records failed: %v", err)
		return nil
	}

	recs := make([]TeacherRecord, 0, len(rows))
	for _, r := range rows {
		recs = append(recs, TeacherRecord{
			ID:       r.MasjidTeacherID,
			MasjidID: r.MasjidTeacherMasjidID,
		})
	}
	return recs
}

/* ==========================
   ISSUE TOKENS + Response
========================== */

func issueTokensWithRoles(
	c *fiber.Ctx,
	db *gorm.DB,
	user userModel.UserModel,
	masjidAdminIDs []string,
	masjidTeacherIDs []string,
	masjidStudentIDs []string,
	masjidIDs []string,
) error {
	// secrets
	jwtSecret, err := getJWTSecret()
	if err != nil {
		return err
	}
	refreshSecret, err := getRefreshSecret()
	if err != nil {
		return err
	}

	now := nowUTC()

	// (baru) teacher_records bila role teacher
	teacherRecords := fetchTeacherRecords(db, user)

	// ACCESS TOKEN
	accessClaims := jwt.MapClaims{
		"typ":                "access",
		"sub":                user.ID.String(),
		"id":                 user.ID.String(),
		"user_name":          user.UserName,
		"full_name":          user.FullName,
		"masjid_admin_ids":   masjidAdminIDs,
		"masjid_teacher_ids": masjidTeacherIDs,
		"masjid_student_ids": masjidStudentIDs,
		"masjid_ids":         masjidIDs,
		"iat":                now.Unix(),
		"exp":                now.Add(accessTTLDefault).Unix(),
	}
	if len(teacherRecords) > 0 {
		accessClaims["teacher_records"] = teacherRecords
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(jwtSecret))
	if err != nil {
		return helpers.Error(c, fiber.StatusInternalServerError, "Gagal membuat access token")
	}

	// REFRESH TOKEN
	refreshClaims := jwt.MapClaims{
		"typ": "refresh",
		"sub": user.ID.String(),
		"id":  user.ID.String(),
		"iat": now.Unix(),
		"exp": now.Add(refreshTTLDefault).Unix(),
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString([]byte(refreshSecret))
	if err != nil {
		return helpers.Error(c, fiber.StatusInternalServerError, "Gagal membuat refresh token")
	}

	// Simpan refresh token hashed
	tokenHash := computeRefreshHash(refreshToken, refreshSecret)
	ua := c.Get("User-Agent")
	ip := c.IP()

	if err := authRepo.CreateRefreshToken(db, &authModel.RefreshTokenModel{
		UserID:    user.ID,
		Token:     tokenHash, // simpan hash
		ExpiresAt: now.Add(refreshTTLDefault),
		UserAgent: strptr(ua),
		IP:        strptr(ip),
	}); err != nil {
		return helpers.Error(c, fiber.StatusInternalServerError, "Gagal menyimpan refresh token")
	}

	// Set cookies
	setAuthCookies(c, accessToken, refreshToken, now)

	// Response body
	respUser := fiber.Map{
		"id":                 user.ID,
		"user_name":          user.UserName,
		"email":              user.Email,
		"full_name":          user.FullName,
		"masjid_admin_ids":   masjidAdminIDs,
		"masjid_teacher_ids": masjidTeacherIDs,
		"masjid_student_ids": masjidStudentIDs,
		"masjid_ids":         masjidIDs,
	}
	if len(teacherRecords) > 0 {
		respUser["teacher_records"] = teacherRecords
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code":    200,
		"status":  "success",
		"message": "Login berhasil",
		"data": fiber.Map{
			"access_token": accessToken,
			"user":         respUser,
		},
	})
}

func setAuthCookies(c *fiber.Ctx, accessToken, refreshToken string, now time.Time) {
	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "None",
		Path:     "/",
		Expires:  now.Add(accessTTLDefault),
	})
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken, // plaintext ke client
		HTTPOnly: true,
		Secure:   true,
		SameSite: "None",
		Path:     "/",
		Expires:  now.Add(refreshTTLDefault),
	})
}

/* ==========================
   LOGIN GOOGLE
========================== */

func LoginGoogle(db *gorm.DB, c *fiber.Ctx) error {
	var input struct {
		IDToken string `json:"id_token"`
	}
	if err := c.BodyParser(&input); err != nil {
		return helpers.Error(c, fiber.StatusBadRequest, "Invalid request body")
	}

	// Verifikasi token Google
	v := googleAuthIDTokenVerifier.Verifier{}
	if err := v.VerifyIDToken(input.IDToken, []string{configs.GoogleClientID}); err != nil {
		return helpers.Error(c, fiber.StatusUnauthorized, "Invalid Google ID Token")
	}

	// Decode claim
	claimSet, err := googleAuthIDTokenVerifier.Decode(input.IDToken)
	if err != nil {
		return helpers.Error(c, fiber.StatusInternalServerError, "Failed to decode ID Token")
	}
	email, name, googleID := claimSet.Email, claimSet.Name, claimSet.Sub

	// Cari by google_id
	user, err := authRepo.FindUserByGoogleID(db, googleID)
	if err != nil {
		// User belum ada -> buat baru
		newUser := userModel.UserModel{
			UserName:         name,
			Email:            email,
			Password:         generateDummyPassword(),
			GoogleID:         &googleID,
			SecurityQuestion: "Created by Google",
			SecurityAnswer:   "google_auth",
			CreatedAt:        nowUTC(),
			UpdatedAt:        nowUTC(),
			IsActive:         true,
		}
		if err := authRepo.CreateUser(db, &newUser); err != nil {
			low := strings.ToLower(err.Error())
			if strings.Contains(low, "duplicate key") || strings.Contains(low, "unique") {
				return helpers.Error(c, fiber.StatusBadRequest, "Email already registered")
			}
			return helpers.Error(c, fiber.StatusInternalServerError, "Failed to create Google user")
		}

		if hasTable(db, "user_progress") {
			_ = progressUserService.CreateInitialUserProgress(db, newUser.ID)
		}
		if hasTable(db, "users_profile") {
			userProfileService.CreateInitialUserProfile(db, newUser.ID)
		}

		user = &newUser
	}

	// Full user + guard aktif
	userFull, err := authRepo.FindUserByID(db, user.ID)
	if err != nil {
		return helpers.Error(c, fiber.StatusInternalServerError, "Gagal mengambil data user")
	}
	if !userFull.IsActive {
		return helpers.Error(c, fiber.StatusForbidden, "Akun Anda telah dinonaktifkan. Hubungi admin.")
	}

	// Peran per masjid
	masjidAdminIDs, masjidTeacherIDs, masjidStudentIDs, masjidIDs, err := collectMasjidRoleIDs(db, userFull.ID)
	if err != nil {
		return helpers.Error(c, fiber.StatusInternalServerError, err.Error())
	}

	// Issue tokens
	return issueTokensWithRoles(c, db, *userFull, masjidAdminIDs, masjidTeacherIDs, masjidStudentIDs, masjidIDs)
}

/* ==========================
   LOGOUT
========================== */

func Logout(db *gorm.DB, c *fiber.Ctx) error {
	// CSRF wajib jika auth via cookie (tanpa Bearer)
	cookieAT := strings.TrimSpace(c.Cookies("access_token"))
	authHeader := strings.TrimSpace(c.Get("Authorization"))
	usesCookieAuth := cookieAT != "" && !strings.HasPrefix(authHeader, "Bearer ")

	if usesCookieAuth {
		if err := helpers.CheckCSRFCookieHeader(c); err != nil {
			return err
		}
	}

	// Ambil raw access token (cookie/Authorization)
	accessToken := helpers.GetRawAccessToken(c)

	// Hitung TTL blacklist
	ttl := resolveBlacklistTTL(accessToken)

	// Blacklist access token (idempotent)
	if accessToken != "" {
		if err := authRepo.BlacklistToken(db, accessToken, ttl); err != nil {
			log.Printf("[WARN] Failed to blacklist token: %v", err)
		}
	} else {
		log.Println("[INFO] Logout tanpa access token; lanjut clear cookies (idempotent)")
	}

	// Hapus refresh token dari DB jika ada di cookie
	if rt := helpers.GetRefreshTokenFromCookie(c); rt != "" {
		// Catatan: jika repo DeleteRefreshToken menerima plaintext dan melakukan hash internal, cukup ini:
		_ = authRepo.DeleteRefreshToken(db, rt)

		// Jika kamu juga menyediakan opsi hapus by hash (mis. DeleteRefreshTokenHashed),
		// aktifkan fallback di bawah:
		// if refreshSecret, err := getRefreshSecret(); err == nil {
		// 	_ = authRepo.DeleteRefreshTokenHashed(db, computeRefreshHash(rt, refreshSecret))
		// }
	}

	// Hapus cookies
	expired := nowUTC().Add(-time.Hour)
	for _, name := range []string{"access_token", "refresh_token", "csrf_token"} {
		c.Cookie(&fiber.Cookie{
			Name:     name,
			Value:    "",
			HTTPOnly: name != "csrf_token",
			Secure:   true,
			SameSite: "None",
			Path:     "/",
			Expires:  expired,
			MaxAge:   -1,
		})
	}

	return helpers.Success(c, "Logout successful", nil)
}

func resolveBlacklistTTL(accessToken string) time.Duration {
	// default dev/test
	ttl := 2 * time.Minute

	// override via env
	if v := os.Getenv("BLACKLIST_TTL_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return time.Duration(n) * time.Second
		}
	}

	// produksi: pakai sisa umur JWT + buffer
	jwtSecret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if jwtSecret == "" || accessToken == "" {
		return ttl
	}
	if tok, err := jwt.Parse(accessToken, func(t *jwt.Token) (any, error) {
		return []byte(jwtSecret), nil
	}); err == nil {
		if claims, ok := tok.Claims.(jwt.MapClaims); ok && tok.Valid {
			if exp, ok := claims["exp"].(float64); ok {
				until := time.Until(time.Unix(int64(exp), 0))
				switch {
				case until > 0:
					return until + 60*time.Second
				default:
					return time.Minute
				}
			}
		}
	}
	return ttl
}

/* ==========================
   UTIL
========================== */

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

	if !strings.EqualFold(strings.TrimSpace(input.Answer), strings.TrimSpace(user.SecurityAnswer)) {
		return helpers.Error(c, fiber.StatusBadRequest, "Incorrect security answer")
	}

	return helpers.Success(c, "Security answer correct", fiber.Map{
		"email": user.Email,
	})
}
