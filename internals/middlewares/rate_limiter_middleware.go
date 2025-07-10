package middlewares

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// Global limiter: untuk semua endpoint biasa
func GlobalRateLimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        100,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"message": "❌ Terlalu banyak permintaan. Silakan coba lagi nanti.",
			})
		},
	})
}

// Rate limiter untuk login route (lebih ketat)
func LoginRateLimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        5,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"message": "❌ Terlalu banyak percobaan login. Coba beberapa saat lagi.",
			})
		},
	})
}

// Rate limiter untuk register route
func RegisterRateLimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        3,
		Expiration: 5 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"message": "❌ Terlalu banyak percobaan pendaftaran. Tunggu beberapa menit ya.",
			})
		},
	})
}

// Rate limiter untuk forgot-password
func ForgotPasswordRateLimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        2,
		Expiration: 10 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"message": "❌ Terlalu banyak permintaan reset password. Silakan coba lagi dalam 10 menit.",
			})
		},
	})
}
