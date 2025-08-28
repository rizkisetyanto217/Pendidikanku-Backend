// internals/middlewares/auth/auth_middleware.go
package auth

import (
	"errors"
	"log"
	"strings" // ⬅️ penting: buat cek prefix /public
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"gorm.io/gorm"

	"masjidku_backend/internals/configs"
	TokenBlacklistModel "masjidku_backend/internals/features/users/auth/model"
)

// Public/webhook path yang di-skip auth strict
var skipPaths = map[string]struct{}{
	"/api/donations/notification": {},
}

const (
	logPrefix         = "[AUTH]"
	defaultExpirySkew = 30 * time.Second // toleransi exp
)

// AuthMiddleware: WAJIB token (untuk /api/u, /api/a). JANGAN pasang untuk /public/*.
func AuthMiddleware(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// ⛔ Jangan verifikasi untuk /public/*
		if strings.HasPrefix(c.Path(), "/public/") {
			return c.Next()
		}

		log.Printf("%s 🔥 %s %s", logPrefix, c.Method(), c.OriginalURL())

		// 1) Skip path tertentu (webhook dsb.)
		if _, ok := skipPaths[c.Path()]; ok {
			log.Printf("%s [INFO] Skip Auth for: %s", logPrefix, c.Path())
			return c.Next()
		}

		// 2) Ambil Authorization (atau cookie) — STRICT
		tokenString, err := extractBearerToken(c)
		if err != nil {
			// Missing token di protected route itu normal → pakai WARN, bukan ERROR
			log.Printf("%s [WARN] missing/invalid token on protected route: %v", logPrefix, err)
			return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
		}

		// 3) Cek blacklist (sekali per request)
		if c.Locals("token_checked") == nil {
			var existing TokenBlacklistModel.TokenBlacklistModel
			if err := db.Where("token = ? AND deleted_at IS NULL", tokenString).First(&existing).Error; err == nil {
				log.Printf("%s [WARN] token is blacklisted", logPrefix)
				return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				log.Printf("%s [ERROR] DB error saat cek blacklist: %v", logPrefix, err)
				return fiber.NewError(fiber.StatusInternalServerError, "Internal Server Error")
			}
			c.Locals("token_checked", true)
		}

		// 4) Parse & verifikasi JWT (tanpa validate claims tambahan)
		secretKey := strings.TrimSpace(configs.JWTSecret)
		if secretKey == "" {
			log.Printf("%s [ERROR] JWT_SECRET kosong", logPrefix)
			return fiber.NewError(fiber.StatusInternalServerError, "Missing JWT Secret")
		}

		claims := jwt.MapClaims{}
		parser := jwt.Parser{SkipClaimsValidation: true}
		if _, err := parser.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			// optional: tegaskan algoritma HMAC
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.ErrUnauthorized
			}
			return []byte(secretKey), nil
		}); err != nil {
			log.Printf("%s [WARN] token parse error: %v", logPrefix, err)
			return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
		}

		// 5) Validasi exp (grace)
		if err := validateTokenExpiry(claims, defaultExpirySkew); err != nil {
			log.Printf("%s [WARN] token expired: %v", logPrefix, err)
			return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
		}

		// 6) Ambil user_id & validasi user aktif
		userID, err := extractUserID(claims)
		if err != nil {
			log.Printf("%s [WARN] invalid/missing user_id: %v", logPrefix, err)
			return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
		}
		c.Locals("user_id", userID.String())

		if err := ensureUserActive(db, userID); err != nil {
			log.Printf("%s [WARN] ensureUserActive: %v", logPrefix, err)
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
			}
			return fiber.NewError(fiber.StatusForbidden, "Akun Anda telah dinonaktifkan")
		}

		// 7) Simpan info klaim ke context (role, user_name, masjid IDs)
		storeBasicClaimsToLocals(c, claims)
		storeMasjidIDsToLocals(c, claims)

		log.Printf("%s [SUCCESS] token valid", logPrefix)
		return c.Next()
	}
}
