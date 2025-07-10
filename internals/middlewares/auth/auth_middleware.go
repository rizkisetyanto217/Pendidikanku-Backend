package auth

import (
	"errors"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"

	"masjidku_backend/internals/configs"
	TokenBlacklistModel "masjidku_backend/internals/features/users/auth/model"

	"gorm.io/gorm"
)

func AuthMiddleware(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		log.Println("🔥 AuthMiddleware triggered at:", c.Path())

		log.Printf("[MIDDLEWARE] Request: %s %s", c.Method(), c.OriginalURL())

		// 🚨 Skip auth untuk Midtrans webhook
		if c.Path() == "/api/donations/notification" {
			log.Println("[INFO] Skip AuthMiddleware untuk webhook Midtrans")
			return c.Next()
		}

		authHeader := c.Get("Authorization")
		log.Println("[DEBUG] Authorization Header:", authHeader)
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized - No token provided",
			})
		}

		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized - Invalid token format",
			})
		}

		tokenString := tokenParts[1]

		// ✅ Optimasi: hindari query berulang ke blacklist
		if c.Locals("token_checked") == nil {
			var existingToken TokenBlacklistModel.TokenBlacklist
			err := db.Where("token = ? AND deleted_at IS NULL", tokenString).First(&existingToken).Error
			if err == nil {
				log.Println("[WARNING] Token ditemukan di blacklist")
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Unauthorized - Token is blacklisted",
				})
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				log.Println("[ERROR] DB error saat cek blacklist:", err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal Server Error",
				})
			}
			c.Locals("token_checked", true)
		}

		// 🔐 Validasi token
		secretKey := configs.JWTSecret
		if secretKey == "" {
			log.Println("[ERROR] JWT_SECRET kosong")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Missing JWT Secret",
			})
		}

		claims := jwt.MapClaims{}
		parser := jwt.Parser{SkipClaimsValidation: true}

		_, err := parser.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(secretKey), nil
		})
		if err != nil {
			log.Println("[ERROR] Gagal parse token:", err)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized - Token parse error",
			})
		}

		// ⏳ Validasi exp token
		exp, exists := claims["exp"].(float64)
		if !exists {
			log.Println("[ERROR] Token tidak memiliki exp")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized - Token has no expiration",
			})
		}

		now := time.Now()
		expTime := time.Unix(int64(exp), 0)
		toleransi := 30 * time.Second
		expired := now.After(expTime.Add(toleransi))

		log.Printf("[DEBUG] now      : %v (Unix: %d)", now, now.Unix())
		log.Printf("[DEBUG] expTime  : %v (Unix: %d)", expTime, int64(exp))
		log.Printf("[DEBUG] expired? : %v (toleransi %v)", expired, toleransi)

		if expired {
			log.Println("[ERROR] Token sudah expired")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized - Token expired",
			})
		}

		// 🧾 Ambil dan simpan user ID dari token
		idStr, exists := claims["id"].(string)
		if !exists {
			log.Println("[ERROR] Token tidak berisi user ID")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized - No user ID in token",
			})
		}
		userID, err := uuid.Parse(idStr)
		if err != nil {
			log.Println("[ERROR] Gagal parse UUID:", err)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized - Invalid user ID format",
			})
		}
		c.Locals("user_id", userID.String())
		log.Println("[SUCCESS] User ID stored:", userID)

		// 🧍 Validasi apakah user aktif
		var user struct {
			IsActive bool
		}
		if err := db.Table("users").Select("is_active").Where("id = ?", userID).First(&user).Error; err != nil {
			log.Println("[ERROR] User tidak ditemukan:", err)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized - User not found",
			})
		}
		if !user.IsActive {
			log.Println("[ERROR] User nonaktif")
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Akun Anda telah dinonaktifkan",
			})
		}

		// 🎭 Simpan role dan nama user dari token ke context
		if role, ok := claims["role"].(string); ok {
			c.Locals("userRole", role)
		}
		if userName, ok := claims["user_name"].(string); ok {
			c.Locals("user_name", userName)
		}

		// 🕌 Simpan daftar masjid_admin_ids ke context (jika ada)
		if masjidIDs, ok := claims["masjid_admin_ids"].([]interface{}); ok {
			var ids []string
			for _, id := range masjidIDs {
				if s, ok := id.(string); ok {
					ids = append(ids, s)
				}
			}
			c.Locals("masjid_admin_ids", ids)
			log.Println("[SUCCESS] Masjid admin IDs stored in context:", ids)
		} else {
			log.Println("[INFO] Token tidak mengandung masjid_admin_ids")
		}

		log.Println("[SUCCESS] Token valid, lanjutkan request")
		return c.Next()
	}
}
