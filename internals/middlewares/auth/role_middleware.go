package auth

import (
	"log"

	"github.com/gofiber/fiber/v2"
)

// RoleMiddlewareWithCustomError validasi role + custom error message
func RoleMiddlewareWithCustomError(allowedRoles []string, customForbiddenMessage string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Ambil role dari context (HARUS seragam)
		role, ok := c.Locals("userRole").(string) // 🔥 pastikan "userRole"
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "Unauthorized: missing role information",
			})
		}

		// 🔥 Tambahkan Log
		log.Printf("[DEBUG] Role pengguna: %s\n", role)

		// Cek apakah role user termasuk allowedRoles
		for _, allowed := range allowedRoles {
			if role == allowed {
				return c.Next()
			}
		}

		// Kalau role tidak cocok
		if customForbiddenMessage == "" {
			customForbiddenMessage = "Forbidden: you are not authorized to access this resource"
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": customForbiddenMessage,
		})
	}
}

// Shortcut biar lebih clean pemakaian
func OnlyRoles(customMessage string, roles ...string) fiber.Handler {
	return RoleMiddlewareWithCustomError(roles, customMessage)
}
