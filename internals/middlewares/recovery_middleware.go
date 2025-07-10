package middlewares

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// RecoveryMiddleware menangkap panic dan mengembalikan error 500
func RecoveryMiddleware() fiber.Handler {
	return recover.New(recover.Config{
		EnableStackTrace: true, // Stack trace akan dicetak saat error
	})
}
