package auth

import (
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

func SecondAuthMiddleware(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		log.Println("🔥 SecondAuthMiddleware triggered at:", c.Path())

		authHeader := c.Get("Authorization")
		if authHeader == "" {
			cookieToken := c.Cookies("access_token")
			if cookieToken != "" {
				authHeader = "Bearer " + cookieToken
			}
		}

		// Jika tetap tidak ada token, lanjutkan tanpa user context
		if authHeader == "" {
			log.Println("[INFO] Tidak ada token, lanjut sebagai anonymous")
			return c.Next()
		}

		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			log.Println("[WARNING] Format token tidak valid, lanjut sebagai anonymous")
			return c.Next()
		}

		tokenString := tokenParts[1]

		// Cek blacklist
		var existingToken TokenBlacklistModel.TokenBlacklist
		if err := db.Where("token = ? AND deleted_at IS NULL", tokenString).First(&existingToken).Error; err == nil {
			log.Println("[WARNING] Token ada di blacklist, lanjut sebagai anonymous")
			return c.Next()
		}

		// Parse & validasi token
		secretKey := configs.JWTSecret
		if secretKey == "" {
			log.Println("[ERROR] JWT_SECRET kosong, lanjut sebagai anonymous")
			return c.Next()
		}

		claims := jwt.MapClaims{}
		parser := jwt.Parser{SkipClaimsValidation: true}

		_, err := parser.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(secretKey), nil
		})
		if err != nil {
			log.Println("[ERROR] Gagal parse token, lanjut sebagai anonymous:", err)
			return c.Next()
		}

		// Validasi exp
		if exp, ok := claims["exp"].(float64); ok {
			expTime := time.Unix(int64(exp), 0)
			if time.Now().After(expTime.Add(30 * time.Second)) {
				log.Println("[WARNING] Token expired, lanjut sebagai anonymous")
				return c.Next()
			}
		}

		// Ambil user ID
		idStr, ok := claims["id"].(string)
		if !ok {
			log.Println("[WARNING] Token tidak memiliki ID, lanjut sebagai anonymous")
			return c.Next()
		}
		userID, err := uuid.Parse(idStr)
		if err != nil {
			log.Println("[WARNING] ID token bukan UUID, lanjut sebagai anonymous")
			return c.Next()
		}

		// Validasi user aktif
		var user struct {
			IsActive bool
		}
		if err := db.Table("users").Select("is_active").Where("id = ?", userID).First(&user).Error; err != nil || !user.IsActive {
			log.Println("[WARNING] User tidak ditemukan atau nonaktif, lanjut sebagai anonymous")
			return c.Next()
		}

		// Simpan user context
		c.Locals("user_id", userID.String())
		if role, ok := claims["role"].(string); ok {
			c.Locals("userRole", role)
		}
		if userName, ok := claims["user_name"].(string); ok {
			c.Locals("user_name", userName)
		}
		if masjidIDs, ok := claims["masjid_admin_ids"].([]interface{}); ok {
			var ids []string
			for _, id := range masjidIDs {
				if s, ok := id.(string); ok {
					ids = append(ids, s)
				}
			}
			c.Locals("masjid_admin_ids", ids)
			if len(ids) > 0 {
				c.Locals("masjid_id", ids[0])
			}
		}

		log.Println("[SUCCESS] Token valid, lanjut sebagai user:", userID)
		return c.Next()
	}
}
