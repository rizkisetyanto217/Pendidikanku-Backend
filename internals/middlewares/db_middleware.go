package middlewares

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// DBMiddleware untuk menambahkan koneksi db ke context request
func DBMiddleware(db *gorm.DB) fiber.Handler {
    return func(c *fiber.Ctx) error {
        // Menyimpan koneksi db di dalam context request
        c.Locals("db", db)
        return c.Next()
    }
}
