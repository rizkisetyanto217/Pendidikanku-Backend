// internals/features/users/auth/service/token_service.go
package service

import (
	"log"
	"strings"
	"time"

	authModel "schoolku_backend/internals/features/users/auth/model"
	authRepo "schoolku_backend/internals/features/users/auth/repository"
	helpers "schoolku_backend/internals/helpers"
	helpersAuth "schoolku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ========================== REFRESH TOKEN ==========================
// POST /api/auth/refresh-token
func RefreshToken(db *gorm.DB, c *fiber.Ctx) error {
	// CSRF wajib untuk endpoint cookie-based
	if err := enforceCSRF(c); err != nil {
		return helpers.JsonError(c, fiber.StatusForbidden, err.Error())
	}
	refreshCookie := strings.TrimSpace(c.Cookies("refresh_token"))
	if refreshCookie == "" {
		return helpers.JsonError(c, fiber.StatusUnauthorized, "Refresh token tidak ada")
	}

	refreshSecret, err := getRefreshSecret()
	if err != nil {
		return helpers.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Parse & validate refresh JWT
	tok, err := jwt.Parse(refreshCookie, func(t *jwt.Token) (any, error) {
		return []byte(refreshSecret), nil
	})
	if err != nil || !tok.Valid {
		return helpers.JsonError(c, fiber.StatusUnauthorized, "Refresh token invalid")
	}
	claims, _ := tok.Claims.(jwt.MapClaims)
	sub, _ := claims["sub"].(string)
	if _, err := uuid.Parse(sub); err != nil {
		return helpers.JsonError(c, fiber.StatusUnauthorized, "Refresh token invalid")
	}
	userID, _ := uuid.Parse(sub)

	// Pastikan hash refresh ada di DB
	h := computeRefreshHash(refreshCookie, refreshSecret)
	var exists bool
	if err := db.Raw(`SELECT EXISTS(SELECT 1 FROM refresh_tokens WHERE token = ?)`, h).Scan(&exists).Error; err != nil {
		return helpers.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}
	if !exists {
		return helpers.JsonError(c, fiber.StatusUnauthorized, "Refresh token tidak dikenal")
	}

	// Ambil user + roles
	userFull, err := authRepo.FindUserByID(db, userID)
	if err != nil {
		return helpers.JsonError(c, fiber.StatusInternalServerError, "User not found")
	}
	if !userFull.IsActive {
		return helpers.JsonError(c, fiber.StatusForbidden, "Akun dinonaktifkan")
	}
	rolesClaim, err := getUserRolesClaim(c.Context(), db, userFull.ID)
	if err != nil {
		return helpers.JsonError(c, fiber.StatusInternalServerError, "Gagal ambil roles")
	}

	// ROTATE: hapus token lama
	if err := deleteRefreshTokenByHash(c.Context(), db, refreshCookie); err != nil {
		log.Printf("[refresh] delete old hash failed: %v", err)
	}

	// issue access & refresh baru (re-use logic tanpa set cookie access)
	jwtSecret, _ := getJWTSecret()
	now := nowUTC()

	// Build claims lagi (ringkas pakai helper lama)
	isOwner := hasGlobalRole(rolesClaim, "owner")
	schoolIDs := deriveSchoolIDsFromRolesClaim(rolesClaim)
	activeSchoolID := helpersAuth.GetActiveSchoolIDIfSingle(rolesClaim)
	teacherRecords := buildTeacherRecords(db, userFull.ID, rolesClaim)
	studentRecords := buildStudentRecords(db, userFull.ID, rolesClaim)
	tpMap := getTenantProfilesMapStr(c.Context(), db, schoolUUIDsFromClaim(rolesClaim))
	combined := combineRolesWithTenant(rolesClaim, tpMap)

	var tenantProfile *string
	if activeSchoolID != nil {
		if mid, err := uuid.Parse(*activeSchoolID); err == nil {
			tenantProfile = getSchoolTenantProfileStr(c.Context(), db, mid)
		}
	}
	accessClaims := buildAccessClaims(*userFull, rolesClaim, schoolIDs, isOwner, activeSchoolID, tenantProfile, combined, teacherRecords, studentRecords, now)
	refreshClaims := buildRefreshClaims(userFull.ID, now)

	newAccess, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(jwtSecret))
	if err != nil {
		return helpers.JsonError(c, fiber.StatusInternalServerError, "Gagal buat access baru")
	}
	newRefresh, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString([]byte(refreshSecret))
	if err != nil {
		return helpers.JsonError(c, fiber.StatusInternalServerError, "Gagal buat refresh baru")
	}

	// simpan hash refresh baru
	if err := createRefreshTokenFast(db, &authModel.RefreshTokenModel{
		UserID:    userFull.ID,
		Token:     computeRefreshHash(newRefresh, refreshSecret),
		ExpiresAt: now.Add(refreshTTLDefault),
		UserAgent: strptr(c.Get("User-Agent")),
		IP:        strptr(c.IP()),
	}); err != nil {
		return helpers.JsonError(c, fiber.StatusInternalServerError, "Gagal simpan refresh baru")
	}

	// set cookie refresh baru + XSRF baru
	setAuthCookiesOnlyRefreshAndXsrf(c, newRefresh, now)

	return helpers.JsonOK(c, "Token diperbarui", fiber.Map{
		"access_token": newAccess,
	})
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

// CSRF: seed cookie XSRF-TOKEN untuk double-submit strategy
func CSRF(db *gorm.DB, c *fiber.Ctx) error {
	origin := getRequestOrigin(c)
	// 🔧 relaks aturan: jika origin kosong, tetap lolos (akan tetap dibatasi oleh CORS layer)
	if origin != "" && !isAllowedOrigin(origin) {
		return helpers.JsonError(c, fiber.StatusForbidden, "Origin not allowed")
	}

	token := randomString(48)
	setXSRFCookie(c, token, nowUTC().Add(24*time.Hour))

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": fiber.Map{"csrf_token": token},
	})
}
