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




func AuthMiddleware(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		log.Printf("🔥 AuthMiddleware: %s %s", c.Method(), c.OriginalURL())

		// 1) Skip path tertentu (webhook dsb.)
		if _, ok := skipPaths[c.Path()]; ok {
			log.Println("[INFO] Skip AuthMiddleware for:", c.Path())
			return c.Next()
		}

		// 2) Ambil Authorization (atau cookie)
		tokenString, err := extractBearerToken(c)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, err.Error())
		}

		// 3) Cek blacklist (sekali per request)
		if c.Locals("token_checked") == nil {
			var existing TokenBlacklistModel.TokenBlacklist
			if err := db.Where("token = ? AND deleted_at IS NULL", tokenString).First(&existing).Error; err == nil {
				log.Println("[WARNING] Token ditemukan di blacklist")
				return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized - Token is blacklisted")
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				log.Println("[ERROR] DB error saat cek blacklist:", err)
				return fiber.NewError(fiber.StatusInternalServerError, "Internal Server Error")
			}
			c.Locals("token_checked", true)
		}

		// 4) Parse & verifikasi JWT (tanpa validate claims tambahan)
		secretKey := configs.JWTSecret
		if secretKey == "" {
			log.Println("[ERROR] JWT_SECRET kosong")
			return fiber.NewError(fiber.StatusInternalServerError, "Missing JWT Secret")
		}

		claims := jwt.MapClaims{}
		parser := jwt.Parser{SkipClaimsValidation: true}
		if _, err := parser.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(secretKey), nil
		}); err != nil {
			log.Println("[ERROR] Gagal parse token:", err)
			return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized - Token parse error")
		}

		// 5) Validasi exp
		if err := validateTokenExpiry(claims, 30*time.Second); err != nil {
			log.Println("[ERROR] Exp validation:", err)
			return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized - Token expired")
		}

		// 6) Ambil user_id & validasi user aktif
		userID, err := extractUserID(claims)
		if err != nil {
			log.Println("[ERROR] user_id:", err)
			return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized - Invalid or missing user ID")
		}
		c.Locals("user_id", userID.String())

		if err := ensureUserActive(db, userID); err != nil {
			log.Println("[ERROR] ensureUserActive:", err)
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized - User not found")
			}
			return fiber.NewError(fiber.StatusForbidden, "Akun Anda telah dinonaktifkan")
		}

		// 7) Simpan info klaim ke context (role, user_name, dan masjid IDs)
		storeBasicClaimsToLocals(c, claims)
		storeMasjidIDsToLocals(c, claims) // <- simpan admin/teacher/union + set masjid_id aktif

		log.Println("[SUCCESS] Token valid, lanjutkan request")
		return c.Next()
	}
}
