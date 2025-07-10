package auth

import (
	"log"
	"masjidku_backend/internals/configs"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"gorm.io/gorm"
)

func OptionalJWTMiddleware(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			return c.Next()
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		secretKey := configs.JWTSecret
		if secretKey == "" {
			log.Println("[ERROR] JWT_SECRET kosong")
			return c.Next()
		}

		claims := jwt.MapClaims{}
		parser := jwt.Parser{SkipClaimsValidation: true}

		_, err := parser.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(secretKey), nil
		})
		if err != nil {
			log.Println("[INFO] OptionalJWTMiddleware: token tidak valid, lanjutkan tanpa user_id")
			return c.Next()
		}

		idStr, exists := claims["id"].(string)
		if exists {
			c.Locals("user_id", idStr)
			log.Println("[INFO] OptionalJWTMiddleware: user_id ditemukan:", idStr)
		}

		// Simpan juga role atau nama kalau perlu
		if role, ok := claims["role"].(string); ok {
			c.Locals("userRole", role)
		}
		if userName, ok := claims["user_name"].(string); ok {
			c.Locals("user_name", userName)
		}

		return c.Next()
	}
}
