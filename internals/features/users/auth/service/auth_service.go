package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	googleAuthIDTokenVerifier "github.com/futurenda/google-auth-id-token-verifier"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"masjidku_backend/internals/configs"
	authHelper "masjidku_backend/internals/features/users/auth/helper"
	authModel "masjidku_backend/internals/features/users/auth/model"
	authRepo "masjidku_backend/internals/features/users/auth/repository"
	userModel "masjidku_backend/internals/features/users/user/model"
	userProfileService "masjidku_backend/internals/features/users/user/service"
	helpers "masjidku_backend/internals/helpers"
	helpersAuth "masjidku_backend/internals/helpers/auth"
)

/* ==========================
   Const & Types
========================== */

const (
	accessTTLDefault  = 24 * time.Hour
	refreshTTLDefault = 7 * 24 * time.Hour

	// timeouts untuk query hot path (aman disesuaikan)
	qryTimeoutShort = 800 * time.Millisecond
	qryTimeoutLong  = 1200 * time.Millisecond
)

type TeacherRecord struct {
	ID       uuid.UUID `json:"masjid_teacher_id" gorm:"column:masjid_teacher_id"`
	MasjidID uuid.UUID `json:"masjid_id"        gorm:"column:masjid_teacher_masjid_id"`
}


type StudentRecord struct {
	ID       uuid.UUID `json:"masjid_student_id" gorm:"column:masjid_student_id"`
	MasjidID uuid.UUID `json:"masjid_id"        gorm:"column:masjid_student_masjid_id"`
}


/* ==========================
   Meta schema cache (prewarm)
========================== */

type authMeta struct {
	once sync.Once
	// tables
	HasMasjidAdmins   bool
	HasMasjidTeachers bool
	HasUserClasses    bool
	HasRoles          bool
	HasUserRoles      bool
	// functions
	HasFnGrantRole      bool
	HasFnUserRolesClaim bool
	HasFnRolePriority   bool

	Ready bool
}

var meta authMeta

// Panggil sekali saat app start setelah DB siap: service.PrewarmAuthMeta(db)
func PrewarmAuthMeta(db *gorm.DB) {
	meta.once.Do(func() {
		meta.HasMasjidAdmins = quickHasTable(db, "masjid_admins")
		meta.HasMasjidTeachers = quickHasTable(db, "masjid_teachers")
		meta.HasUserClasses = quickHasTable(db, "user_classes")
		meta.HasRoles = quickHasTable(db, "roles")
		meta.HasUserRoles = quickHasTable(db, "user_roles")
		meta.HasFnGrantRole = quickHasFunction(db, "fn_grant_role")
		meta.HasFnUserRolesClaim = quickHasFunction(db, "fn_user_roles_claim")
		meta.HasFnRolePriority = quickHasFunction(db, "fn_role_priority")
		meta.Ready = true
	})
}

func quickHasTable(db *gorm.DB, table string) bool {
	if db == nil || table == "" {
		return false
	}
	var exists bool
	_ = db.Raw(`SELECT to_regclass((SELECT current_schema()) || '.' || ?) IS NOT NULL`, table).Scan(&exists).Error
	return exists
}

func quickHasFunction(db *gorm.DB, name string) bool {
	if db == nil || name == "" {
		return false
	}
	var ok bool
	_ = db.Raw(`SELECT EXISTS(SELECT 1 FROM pg_proc WHERE proname = ?)`, name).Scan(&ok).Error
	return ok
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
	_, _ = m.Write([]byte(token))
	return m.Sum(nil)
}

// Cek role hanya di roles_global (bukan scoped)
func hasGlobalRole(rc helpersAuth.RolesClaim, role string) bool {
	role = strings.ToLower(role)
	for _, r := range rc.RolesGlobal {
		if strings.EqualFold(r, role) {
			return true
		}
	}
	return false
}


func rolesClaimHas(rc helpersAuth.RolesClaim, role string) bool {
	role = strings.ToLower(role)
	for _, r := range rc.RolesGlobal {
		if strings.EqualFold(r, role) {
			return true
		}
	}
	for _, mr := range rc.MasjidRoles {
		for _, r := range mr.Roles {
			if strings.EqualFold(r, role) {
				return true
			}
		}
	}
	return false
}

// Derive masjid_ids (opsional, untuk kompat) dari masjid_roles.
func deriveMasjidIDsFromRolesClaim(rc helpersAuth.RolesClaim) []string {
	set := map[string]struct{}{}
	for _, mr := range rc.MasjidRoles {
		if mr.MasjidID != uuid.Nil {
			set[mr.MasjidID.String()] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	return out
}

/* ==========================
   REGISTER
========================== */
func Register(db *gorm.DB, c *fiber.Ctx) error {
	var input userModel.UserModel
	if err := c.BodyParser(&input); err != nil {
		return helpers.JsonError(c, fiber.StatusBadRequest, "Invalid request body")
	}

	// ---------- Normalisasi ringan ----------
	input.UserName = strings.TrimSpace(input.UserName)
	input.Email = strings.TrimSpace(strings.ToLower(input.Email))
	if input.FullName != nil {
		f := strings.TrimSpace(*input.FullName)
		input.FullName = &f
	}
	if input.GoogleID != nil {
		g := strings.TrimSpace(*input.GoogleID)
		if g == "" {
			input.GoogleID = nil
		} else {
			input.GoogleID = &g
		}
	}
	if input.Password != nil {
		p := strings.TrimSpace(*input.Password)
		if p == "" {
			input.Password = nil
		} else {
			input.Password = &p
		}
	}

	// ---------- Validasi bisnis: minimal password ATAU google_id ----------
	if (input.Password == nil || *input.Password == "") && (input.GoogleID == nil || *input.GoogleID == "") {
		return helpers.JsonError(c, fiber.StatusBadRequest, "password atau google_id wajib diisi salah satu")
	}

	// ---------- Validasi field sesuai tag di model ----------
	if err := input.Validate(); err != nil {
		return helpers.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// ---------- Hash password (jika ada) ----------
	if input.Password != nil && *input.Password != "" {
		hashed, err := authHelper.HashPassword(*input.Password)
		if err != nil {
			return helpers.JsonError(c, fiber.StatusInternalServerError, "Password hashing failed")
		}
		input.Password = &hashed
	}

	// created_at/updated_at default by DB; is_active default true by DB

	// ---------- Create user ----------
	if err := authRepo.CreateUser(db, &input); err != nil {
		low := strings.ToLower(err.Error())
		switch {
		case strings.Contains(low, "uq_users_email") || strings.Contains(low, "users_email_key") || strings.Contains(low, "email") && strings.Contains(low, "unique"):
			return helpers.JsonError(c, fiber.StatusBadRequest, "Email already registered")
		case strings.Contains(low, "users_user_name") && strings.Contains(low, "unique"):
			return helpers.JsonError(c, fiber.StatusBadRequest, "Username already taken")
		case strings.Contains(low, "uq_users_google_id") || strings.Contains(low, "google_id") && strings.Contains(low, "unique"):
			return helpers.JsonError(c, fiber.StatusBadRequest, "Google account already linked to another user")
		default:
			return helpers.JsonError(c, fiber.StatusInternalServerError, "Failed to create user")
		}
	}

	// ---------- Best-effort init entitas turunan ----------
	if meta.Ready && quickHasTable(db, "users_profile") {
		// CreateInitialUserProfile tidak mengembalikan apa-apa → cukup panggil
		userProfileService.CreateInitialUserProfile(db, input.ID)
	}

	// ---------- Grant default role "user" ----------
	if err := grantDefaultUserRole(c.Context(), db, input.ID); err != nil {
		log.Printf("[register] grant default role 'user' failed: %v", err)
	}

	return helpers.JsonCreated(c, "Registration successful", nil)
}

/* ==========================
   Helpers (role)
========================== */

func grantDefaultUserRole(ctx context.Context, db *gorm.DB, userID uuid.UUID) error {
	if !meta.Ready || !meta.HasRoles || !meta.HasUserRoles {
		return nil
	}

	// Prefer function bila ada
	if meta.HasFnGrantRole {
		var idStr string
		if err := db.WithContext(ctx).
			Raw(`SELECT fn_grant_role(?::uuid, ?::text, NULL::uuid, NULL::uuid)::text`, userID.String(), "user").
			Scan(&idStr).Error; err != nil {
			return err
		}
		if idStr != "" {
			if _, perr := uuid.Parse(idStr); perr != nil {
				log.Printf("[grantDefaultUserRole] warning: parse uuid failed: %v", perr)
			}
		}
		return nil
	}

	// Fallback manual
	var roleIDStr string
	if err := db.WithContext(ctx).
		Raw(`SELECT role_id::text FROM roles WHERE role_name = 'user' LIMIT 1`).
		Scan(&roleIDStr).Error; err != nil {
		return err
	}
	if roleIDStr == "" {
		if err := db.WithContext(ctx).
			Raw(`INSERT INTO roles(role_name) VALUES ('user') RETURNING role_id::text`).
			Scan(&roleIDStr).Error; err != nil {
			return err
		}
	}

	var exists bool
	if err := db.WithContext(ctx).
		Raw(`SELECT EXISTS(
		       SELECT 1 FROM user_roles
		       WHERE user_id = ?::uuid AND role_id = ?::uuid AND masjid_id IS NULL AND deleted_at IS NULL
		     )`, userID.String(), roleIDStr).
		Scan(&exists).Error; err != nil {
		return err
	}
	if exists {
		return nil
	}

	return db.WithContext(ctx).
		Exec(`INSERT INTO user_roles(user_id, role_id, masjid_id, assigned_at)
		      VALUES (?::uuid, ?::uuid, NULL, now())`,
			userID.String(), roleIDStr).Error
}

// Ambil roles via function claim (jika ada) atau fallback query manual
func getUserRolesClaim(ctx context.Context, db *gorm.DB, userID uuid.UUID) (helpersAuth.RolesClaim, error) {
	out := helpersAuth.RolesClaim{
		RolesGlobal: make([]string, 0),
		MasjidRoles: make([]helpersAuth.MasjidRolesEntry, 0),
	}

	// Pakai fn_user_roles_claim bila ada
	if meta.Ready && meta.HasFnUserRolesClaim {
		var jsonStr string
		if err := db.WithContext(ctx).
			Raw(`SELECT fn_user_roles_claim(?::uuid)::text`, userID.String()).
			Scan(&jsonStr).Error; err != nil {
			return out, err
		}
		if strings.TrimSpace(jsonStr) != "" {
			if err := json.Unmarshal([]byte(jsonStr), &out); err != nil {
				return out, err
			}
		}

		// jaga-jaga kalau function kirim null
		if out.RolesGlobal == nil {
			out.RolesGlobal = []string{}
		}
		if out.MasjidRoles == nil {
			out.MasjidRoles = []helpersAuth.MasjidRolesEntry{}
		}

		return out, nil
	}

	// Fallback manual
	orderBy := "r.role_name ASC"
	if meta.Ready && meta.HasFnRolePriority {
		orderBy = "fn_role_priority(r.role_name) DESC, r.role_name ASC"
	}

	// Global
	{
		ctxG, cancel := context.WithTimeout(ctx, qryTimeoutShort)
		defer cancel()
		var globals []string
		if err := db.WithContext(ctxG).Raw(`
			SELECT r.role_name
			FROM user_roles ur
			JOIN roles r ON r.role_id = ur.role_id
			WHERE ur.user_id = ?::uuid
			  AND ur.deleted_at IS NULL
			  AND ur.masjid_id IS NULL
			GROUP BY r.role_name
			ORDER BY `+orderBy, userID.String()).
			Scan(&globals).Error; err != nil {
			return out, err
		}
		out.RolesGlobal = globals
	}

	// Scoped
	var masjidIDs []uuid.UUID
	{
		ctxS, cancel := context.WithTimeout(ctx, qryTimeoutShort)
		defer cancel()
		if err := db.WithContext(ctxS).Raw(`
			SELECT ur.masjid_id
			FROM user_roles ur
			WHERE ur.user_id = ?::uuid
			  AND ur.deleted_at IS NULL
			  AND ur.masjid_id IS NOT NULL
			GROUP BY ur.masjid_id
		`, userID.String()).
			Scan(&masjidIDs).Error; err != nil {
			return out, err
		}
	}
	for _, mid := range masjidIDs {
		ctxR, cancel := context.WithTimeout(ctx, qryTimeoutShort)
		var roles []string
		err := db.WithContext(ctxR).Raw(`
			SELECT r.role_name
			FROM user_roles ur
			JOIN roles r ON r.role_id = ur.role_id
			WHERE ur.user_id = ?::uuid
			  AND ur.deleted_at IS NULL
			  AND ur.masjid_id = ?::uuid
			GROUP BY r.role_name
			ORDER BY `+orderBy, userID.String(), mid.String()).
			Scan(&roles).Error
		cancel()
		if err != nil {
			return out, err
		}
		out.MasjidRoles = append(out.MasjidRoles, helpersAuth.MasjidRolesEntry{
			MasjidID: mid,
			Roles:    roles,
		})
	}
	return out, nil
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
		return helpers.JsonError(c, fiber.StatusBadRequest, "Invalid input format")
	}
	input.Identifier = strings.TrimSpace(input.Identifier)

	if err := authHelper.ValidateLoginInput(input.Identifier, input.Password); err != nil {
		return helpers.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Ambil minimal user (include kolom password)
	userLight, err := authRepo.FindUserByEmailOrUsernameLight(db, input.Identifier)
	if err != nil {
		// Jangan bocorkan apakah identifier valid — balas generik
		return helpers.JsonError(c, fiber.StatusUnauthorized, "Identifier atau Password salah")
	}
	if !userLight.IsActive {
		return helpers.JsonError(c, fiber.StatusForbidden, "Akun Anda telah dinonaktifkan. Hubungi admin.")
	}

	// 🔒 Tolak jika akun tidak punya password (akun SSO/Google-only)
	if userLight.Password == nil || *userLight.Password == "" {
		return helpers.JsonError(c, fiber.StatusUnauthorized, "Akun ini tidak memiliki password. Silakan login dengan Google atau set password terlebih dahulu.")
	}

	// ✅ Cek hash (dereference pointer)
	if err := authHelper.CheckPasswordHash(*userLight.Password, input.Password); err != nil {
		return helpers.JsonError(c, fiber.StatusUnauthorized, "Identifier atau Password salah")
	}

	// Ambil full user
	userFull, err := authRepo.FindUserByID(db, userLight.ID)
	if err != nil {
		return helpers.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data user")
	}

	// Roles (roles_global & masjid_roles)
	rolesClaim, err := getUserRolesClaim(c.Context(), db, userFull.ID)
	if err != nil {
		return helpers.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil roles user")
	}

	// Issue tokens — cukup berdasarkan rolesClaim
	return issueTokensWithRoles(c, db, *userFull, rolesClaim)
}


/* ==========================
   ISSUE TOKENS + Response
========================== */

func fetchStudentRecords(db *gorm.DB, userID uuid.UUID) []StudentRecord {
	// Pastikan meta sudah di-warm
	if !meta.Ready {
		PrewarmAuthMeta(db)
	}

	ctx, cancel := context.WithTimeout(context.Background(), qryTimeoutShort)
	defer cancel()

	var out []StudentRecord
	err := db.WithContext(ctx).
		Table("masjid_students").
		Select("masjid_student_id, masjid_student_masjid_id").
		Where("masjid_student_user_id = ? AND (masjid_student_deleted_at IS NULL)", userID).
		Scan(&out).Error

	if err != nil {
		low := strings.ToLower(err.Error())
		// Kalau tabel belum ada, anggap tidak ada record
		if strings.Contains(low, "does not exist") ||
			strings.Contains(low, "undefined table") ||
			strings.Contains(low, "no such table") {
			return nil
		}
		log.Printf("[WARN] fetchStudentRecords: %v", err)
		return nil
	}

	return out
}


func fetchTeacherRecords(db *gorm.DB, userID uuid.UUID) []TeacherRecord {
	// Pastikan meta sudah di-warm
	if !meta.Ready {
		PrewarmAuthMeta(db)
	}

	ctx, cancel := context.WithTimeout(context.Background(), qryTimeoutShort)
	defer cancel()

	var out []TeacherRecord
	err := db.WithContext(ctx).
		Table("masjid_teachers").
		Select("masjid_teacher_id, masjid_teacher_masjid_id").
		Where("masjid_teacher_user_id = ? AND (masjid_teacher_deleted_at IS NULL)", userID).
		Scan(&out).Error

	if err != nil {
		low := strings.ToLower(err.Error())
		// Kalau tabel belum ada, jangan panik — anggap tidak ada record
		if strings.Contains(low, "does not exist") ||
			strings.Contains(low, "undefined table") ||
			strings.Contains(low, "no such table") {
			return nil
		}
		log.Printf("[WARN] fetchTeacherRecords: %v", err)
		return nil
	}

	return out
}

// ==========================
// Helpers (roles / teacher)
// ==========================

func masjidIDSetFromClaim(rc helpersAuth.RolesClaim) map[uuid.UUID]struct{} {
	set := make(map[uuid.UUID]struct{}, len(rc.MasjidRoles))
	for _, mr := range rc.MasjidRoles {
		if mr.MasjidID != uuid.Nil {
			set[mr.MasjidID] = struct{}{}
		}
	}
	return set
}

// Ambil teacher_records hanya jika user punya role "teacher".
// (Opsional) filter agar hanya masjid yang ada di masjid_roles claim.
func buildTeacherRecords(db *gorm.DB, userID uuid.UUID, rc helpersAuth.RolesClaim) []TeacherRecord {
	if !rolesClaimHas(rc, "teacher") {
		return nil
	}
	list := fetchTeacherRecords(db, userID)
	if len(list) == 0 {
		return nil
	}
	allow := masjidIDSetFromClaim(rc)
	if len(allow) == 0 {
		return list
	}
	out := make([]TeacherRecord, 0, len(list))
	for _, t := range list {
		if _, ok := allow[t.MasjidID]; ok {
			out = append(out, t)
		}
	}
	return out
}

// Ambil student_records hanya jika user punya role "student".
// (Opsional) filter agar hanya masjid yang ada di masjid_roles claim.
func buildStudentRecords(db *gorm.DB, userID uuid.UUID, rc helpersAuth.RolesClaim) []StudentRecord {
	if !rolesClaimHas(rc, "student") {
		return nil
	}
	list := fetchStudentRecords(db, userID)
	if len(list) == 0 {
		return nil
	}
	allow := masjidIDSetFromClaim(rc)
	if len(allow) == 0 {
		return list
	}
	out := make([]StudentRecord, 0, len(list))
	for _, s := range list {
		if _, ok := allow[s.MasjidID]; ok {
			out = append(out, s)
		}
	}
	return out
}



// ==========================
// Helpers (JWT claims & resp)
// ==========================

func buildRefreshClaims(userID uuid.UUID, now time.Time) jwt.MapClaims {
	return jwt.MapClaims{
		"typ": "refresh",
		"sub": userID.String(),
		"id":  userID.String(),
		"iat": now.Unix(),
		"exp": now.Add(refreshTTLDefault).Unix(),
	}
}

// akses token claims builder — pakai *string
func buildAccessClaims(
	user userModel.UserModel,
	rc helpersAuth.RolesClaim,
	masjidIDs []string,
	isOwner bool,
	activeMasjidID *string, // *string
	teacherRecords []TeacherRecord,
	studentRecords []StudentRecord,
	now time.Time,
) jwt.MapClaims {
	claims := jwt.MapClaims{
		"typ":          "access",
		"sub":          user.ID.String(),
		"id":           user.ID.String(),
		"user_name":    user.UserName,
		"full_name":    user.FullName,
		"roles_global": rc.RolesGlobal,
		"masjid_roles": rc.MasjidRoles,
		"masjid_ids":   masjidIDs,
		"is_owner":     isOwner,
		"iat":          now.Unix(),
		"exp":          now.Add(accessTTLDefault).Unix(),
	}
	if len(teacherRecords) > 0 {
		claims["teacher_records"] = teacherRecords
	}
	if activeMasjidID != nil {
		claims["active_masjid_id"] = *activeMasjidID // string langsung
	}
	if len(studentRecords) > 0 {
		claims["student_records"] = studentRecords
	}
	return claims
}

// login response user builder — pakai *string
func buildLoginResponseUser(
	user userModel.UserModel,
	rc helpersAuth.RolesClaim,
	masjidIDs []string,
	isOwner bool,
	activeMasjidID *string, // *string
	teacherRecords []TeacherRecord,
	studentRecords []StudentRecord,
) fiber.Map {
	resp := fiber.Map{
		"id":           user.ID,
		"user_name":    user.UserName,
		"email":        user.Email,
		"full_name":    user.FullName,
		"roles_global": rc.RolesGlobal,
		"masjid_roles": rc.MasjidRoles,
		"masjid_ids":   masjidIDs,
		"is_owner":     isOwner,
	}
	if len(teacherRecords) > 0 {
		resp["teacher_records"] = teacherRecords
	}
	if activeMasjidID != nil {
		resp["active_masjid_id"] = *activeMasjidID // string langsung
	}
	if len(studentRecords) > 0 {
		resp["student_records"] = studentRecords
	}
	return resp
}

// ==========================
// ISSUE TOKENS (refactor)
// ==========================

func issueTokensWithRoles(
	c *fiber.Ctx,
	db *gorm.DB,
	user userModel.UserModel,
	rolesClaim helpersAuth.RolesClaim,
) error {
	// secrets
	jwtSecret, err := getJWTSecret()
	if err != nil {
		return helpers.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	refreshSecret, err := getRefreshSecret()
	if err != nil {
		return helpers.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	now := nowUTC()

	if !meta.Ready {
		PrewarmAuthMeta(db)
	}

	// Derivatives dari claim
	isOwner := hasGlobalRole(rolesClaim, "owner")
	masjidIDs := deriveMasjidIDsFromRolesClaim(rolesClaim)               // kompat opsional
	activeMasjidID := helpersAuth.GetActiveMasjidIDIfSingle(rolesClaim) // autopick aktif
	teacherRecords := buildTeacherRecords(db, user.ID, rolesClaim)       // ambil + filter
	studentRecords := buildStudentRecords(db, user.ID, rolesClaim) // ambil + filter


	// Access & Refresh claims
	accessClaims := buildAccessClaims(user, rolesClaim, masjidIDs, isOwner, activeMasjidID, teacherRecords, studentRecords, now)
	refreshClaims := buildRefreshClaims(user.ID, now)

	// Sign tokens
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(jwtSecret))
	if err != nil {
		return helpers.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat access token")
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString([]byte(refreshSecret))
	if err != nil {
		return helpers.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat refresh token")
	}

	// Simpan refresh token (hashed)
	tokenHash := computeRefreshHash(refreshToken, refreshSecret)
	ua, ip := c.Get("User-Agent"), c.IP()
	if err := createRefreshTokenFast(db, &authModel.RefreshTokenModel{
		UserID:    user.ID,
		Token:     tokenHash,
		ExpiresAt: now.Add(refreshTTLDefault),
		UserAgent: strptr(ua),
		IP:        strptr(ip),
	}); err != nil {
		return helpers.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan refresh token")
	}

	// Cookies
	setAuthCookies(c, accessToken, refreshToken, now)

	// Response
	respUser := buildLoginResponseUser(user, rolesClaim, masjidIDs, isOwner, activeMasjidID, teacherRecords, studentRecords)
	return helpers.JsonOK(c, "Login berhasil", fiber.Map{
		"user":         respUser,
		"access_token": accessToken,
	})
}

// Insert refresh_token dengan latency lebih rendah.
// Aman untuk token (konsekuensi: kemungkinan kecil lose jika crash tepat sesudah commit).
func createRefreshTokenFast(db *gorm.DB, rt *authModel.RefreshTokenModel) error {
	return db.Transaction(func(tx *gorm.DB) error {
		// turunkan sinkronisasi walau cuma untuk transaksi ini
		if err := tx.Exec(`SET LOCAL synchronous_commit = OFF`).Error; err != nil {
			log.Printf("[WARN] set synchronous_commit=OFF failed: %v", err)
		}
		return authRepo.CreateRefreshToken(tx, rt)
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
		Value:    refreshToken,
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
		return helpers.JsonError(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if strings.TrimSpace(input.IDToken) == "" {
		return helpers.JsonError(c, fiber.StatusBadRequest, "id_token is required")
	}

	

	// Verifikasi token Google (audience = client_id aplikasi kita)
	v := googleAuthIDTokenVerifier.Verifier{}
	if err := v.VerifyIDToken(input.IDToken, []string{configs.GoogleClientID}); err != nil {
		return helpers.JsonError(c, fiber.StatusUnauthorized, "Invalid Google ID Token")
	}

	// Decode claim
	claimSet, err := googleAuthIDTokenVerifier.Decode(input.IDToken)
	if err != nil {
		return helpers.JsonError(c, fiber.StatusInternalServerError, "Failed to decode ID Token")
	}
	email := strings.ToLower(strings.TrimSpace(claimSet.Email))
	name := strings.TrimSpace(claimSet.Name)
	googleID := strings.TrimSpace(claimSet.Sub)

	if email == "" || googleID == "" {
		return helpers.JsonError(c, fiber.StatusUnauthorized, "Google token missing required fields")
	}

	// 1) Coba cari user by google_id
	user, err := authRepo.FindUserByGoogleID(db, googleID)
	if err != nil {
		// 2) Tidak ada: coba cari by email
		userByEmail, err2 := authRepo.FindUserByEmail(db, email)
		if err2 == nil && userByEmail != nil {
			// a) Sudah ada akun dengan email tsb → link google_id (kalau belum)
			if userByEmail.GoogleID == nil || *userByEmail.GoogleID == "" {
				now := time.Now().UTC()
				userByEmail.GoogleID = &googleID
				if userByEmail.EmailVerifiedAt == nil {
					userByEmail.EmailVerifiedAt = &now
				}
				if err := db.Model(userByEmail).Select(
					"google_id", "email_verified_at", "updated_at",
				).Updates(map[string]any{
					"google_id":        userByEmail.GoogleID,
					"email_verified_at": userByEmail.EmailVerifiedAt,
					"updated_at":        now,
				}).Error; err != nil {
					return helpers.JsonError(c, fiber.StatusInternalServerError, "Failed to link Google account")
				}
			}
			user = userByEmail
		} else {
			// b) Tidak ada email di DB → buat user baru (tanpa password)
			now := time.Now().UTC()
			fullName := ptrIfNotEmpty(name)

			// Tentukan username yang aman & unik
			baseUsername := suggestUsername(name, email)
			username := baseUsername
			for i := 0; i < 5; i++ { // sampai 5 kali coba unik
				if exists, _ := authRepo.IsUsernameTaken(db, username); !exists {
					break
				}
				username = baseUsername + "-" + shortRand()
			}

			newUser := userModel.UserModel{
				UserName:        username,
				FullName:        fullName,
				Email:           email,
				Password:        nil,            // Google-only (no local password yet)
				GoogleID:        &googleID,
				IsActive:        true,
				EmailVerifiedAt: &now,
				CreatedAt:       now,
				UpdatedAt:       now,
			}
			if err := authRepo.CreateUser(db, &newUser); err != nil {
				low := strings.ToLower(err.Error())
				switch {
				case strings.Contains(low, "uq_users_email") || strings.Contains(low, "users_email_key"):
					return helpers.JsonError(c, fiber.StatusBadRequest, "Email already registered")
				case strings.Contains(low, "users_user_name") && strings.Contains(low, "unique"):
					return helpers.JsonError(c, fiber.StatusBadRequest, "Username already taken")
				case strings.Contains(low, "uq_users_google_id"):
					return helpers.JsonError(c, fiber.StatusBadRequest, "Google account already linked to another user")
				default:
					return helpers.JsonError(c, fiber.StatusInternalServerError, "Failed to create Google user")
				}
			}
			if meta.Ready && quickHasTable(db, "users_profile") {
				userProfileService.CreateInitialUserProfile(db, newUser.ID)
			}

			user = &newUser
		}
	}

	// 3) Ambil full user + guard aktif
	userFull, err := authRepo.FindUserByID(db, user.ID)
	if err != nil {
		return helpers.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data user")
	}
	if !userFull.IsActive {
		return helpers.JsonError(c, fiber.StatusForbidden, "Akun Anda telah dinonaktifkan. Hubungi admin.")
	}

	// 4) Roles (roles_global & masjid_roles)
	rolesClaim, err := getUserRolesClaim(c.Context(), db, userFull.ID)
	if err != nil {
		return helpers.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil roles user")
	}

	// 5) Issue tokens — cukup berdasarkan rolesClaim
	return issueTokensWithRoles(c, db, *userFull, rolesClaim)
}

/* ==========================
   Helpers khusus login Google
========================== */

func ptrIfNotEmpty(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

// suggestUsername: dari nama → slug-ish; fallback ambil bagian local dari email
func suggestUsername(name, email string) string {
	cand := strings.ToLower(strings.TrimSpace(name))
	cand = strings.ReplaceAll(cand, "  ", " ")
	cand = strings.ReplaceAll(cand, " ", "-")
	cand = sanitizeUsername(cand)
	if cand == "" {
		if i := strings.Index(email, "@"); i > 0 {
			cand = sanitizeUsername(email[:i])
		}
	}
	if cand == "" {
		cand = "user"
	}
	if len(cand) > 50 {
		cand = cand[:50]
	}
	return cand
}

// sanitizeUsername: simpan huruf/angka/dash/underscore saja
func sanitizeUsername(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune(r)
		}
	}
	return b.String()
}

func shortRand() string {
	// ringkas: 4 chars hex dari unixnano
	return strconv.FormatInt(time.Now().UnixNano()%0xffff, 16)
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
			return helpers.JsonError(c, fiber.StatusForbidden, err.Error())
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
		_ = authRepo.DeleteRefreshToken(db, rt)
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

	return helpers.JsonOK(c, "Logout successful", nil)
}

func resolveBlacklistTTL(accessToken string) time.Duration {
	ttl := 2 * time.Minute
	if v := os.Getenv("BLACKLIST_TTL_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return time.Duration(n) * time.Second
		}
	}
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
				if until > 0 {
					return until + 60*time.Second
				}
				return time.Minute
			}
		}
	}
	return ttl
}

/* ==========================
   UTIL
========================== */

// func CheckSecurityAnswer(db *gorm.DB, c *fiber.Ctx) error {
// 	return helpers.JsonError(c, fiber.StatusGone, "Security Q/A sudah tidak didukung. Gunakan alur reset password via email OTP atau magic link.")
// }
