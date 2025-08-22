// internals/middlewares/auth/auth_middleware.go
package auth

import (
	"errors"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"gorm.io/gorm"

	"masjidku_backend/internals/configs"
	TokenBlacklistModel "masjidku_backend/internals/features/users/auth/model"
)

// Public webhook path yang di-skip auth
var skipPaths = map[string]struct{}{
	"/api/donations/notification": {},
}

const (
	logPrefix = "[AUTH]"
	// Skew waktu untuk toleransi exp (mis. clock drift)
	defaultExpirySkew = 30 * time.Second
)

// AuthMiddleware memverifikasi JWT, cek blacklist, validasi user aktif,
// lalu memetakan klaim penting (role, masjid_ids, dll) ke fiber.Context.Locals.
func AuthMiddleware(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		log.Printf("%s 🔥 %s %s", logPrefix, c.Method(), c.OriginalURL())

		// 1) Skip path tertentu (webhook dsb.)
		if _, ok := skipPaths[c.Path()]; ok {
			log.Printf("%s [INFO] Skip Auth for: %s", logPrefix, c.Path())
			return c.Next()
		}

		// 2) Ambil Authorization (atau cookie)
		tokenString, err := extractBearerToken(c)
		if err != nil {
			log.Printf("%s [ERROR] extract token: %v", logPrefix, err)
			return fiber.NewError(fiber.StatusUnauthorized, err.Error())
		}

		// 3) Cek blacklist (sekali per request)
		if c.Locals("token_checked") == nil {
			var existing TokenBlacklistModel.TokenBlacklist
			if err := db.Where("token = ? AND deleted_at IS NULL", tokenString).First(&existing).Error; err == nil {
				log.Printf("%s [WARNING] Token ditemukan di blacklist", logPrefix)
				return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized - Token is blacklisted")
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				log.Printf("%s [ERROR] DB error saat cek blacklist: %v", logPrefix, err)
				return fiber.NewError(fiber.StatusInternalServerError, "Internal Server Error")
			}
			c.Locals("token_checked", true)
		}

		// 4) Parse & verifikasi JWT (tanpa validate claims tambahan)
		secretKey := configs.JWTSecret
		if secretKey == "" {
			log.Printf("%s [ERROR] JWT_SECRET kosong", logPrefix)
			return fiber.NewError(fiber.StatusInternalServerError, "Missing JWT Secret")
		}

		claims := jwt.MapClaims{}
		parser := jwt.Parser{SkipClaimsValidation: true}
		if _, err := parser.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(secretKey), nil
		}); err != nil {
			log.Printf("%s [ERROR] Gagal parse token: %v", logPrefix, err)
			return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized - Token parse error")
		}

		// 5) Validasi exp
		if err := validateTokenExpiry(claims, defaultExpirySkew); err != nil {
			log.Printf("%s [ERROR] Exp validation: %v", logPrefix, err)
			return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized - Token expired")
		}

		// 6) Ambil user_id & validasi user aktif
		userID, err := extractUserID(claims)
		if err != nil {
			log.Printf("%s [ERROR] user_id: %v", logPrefix, err)
			return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized - Invalid or missing user ID")
		}
		c.Locals("user_id", userID.String())

		if err := ensureUserActive(db, userID); err != nil {
			log.Printf("%s [ERROR] ensureUserActive: %v", logPrefix, err)
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized - User not found")
			}
			return fiber.NewError(fiber.StatusForbidden, "Akun Anda telah dinonaktifkan")
		}

		// 7) Simpan info klaim ke context (role, user_name, dan masjid IDs)
		//    Implementasi ada di auth_locals.go
		storeBasicClaimsToLocals(c, claims)
		storeMasjidIDsToLocals(c, claims)

		log.Printf("%s [SUCCESS] Token valid, lanjutkan request", logPrefix)
		return c.Next()
	}
}
