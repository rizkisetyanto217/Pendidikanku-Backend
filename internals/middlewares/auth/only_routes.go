package auth

import (
	"github.com/gofiber/fiber/v2"
)

// OnlyRolesSlice memungkinkan akses jika user memiliki salah satu dari role yang diizinkan.
func OnlyRolesSlice(message string, allowedRoles []string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, ok := c.Locals("userRole").(string)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "Unauthorized - Role not found",
			})
		}

		for _, allowed := range allowedRoles {
			if role == allowed {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": message,
		})
	}
}
