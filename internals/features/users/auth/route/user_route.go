package route

import (
	controller "masjidku_backend/internals/features/users/auth/controller"
	rateLimiter "masjidku_backend/internals/middlewares"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AuthRoutes(app *fiber.App, db *gorm.DB) {
	authController := controller.NewAuthController(db)

	// rate limiter global
	app.Use(rateLimiter.GlobalRateLimiter())

	// --- PUBLIC (/auth) ---
	publicAuth := app.Group("/api/auth")
	publicAuth.Post("/login", rateLimiter.LoginRateLimiter(), authController.Login)
	publicAuth.Post("/register", rateLimiter.RegisterRateLimiter(), authController.Register)
	publicAuth.Post("/forgot-password/reset", authController.ResetPassword)
	// publicAuth.Post("/login-google", authController.LoginGoogle)

	// ⬇️⬇️ Tambahkan ini:
	publicAuth.Get("/csrf", authController.CSRF)

	publicAuth.Post("/refresh-token", authController.RefreshToken)

	// --- PROTECTED (/api/auth ...) ---
	protectedAuth := app.Group("/api/auth")
	protectedAuth.Post("/logout", authController.Logout)
	protectedAuth.Post("/change-password", authController.ChangePassword)
	protectedAuth.Put("/update-user-name", authController.UpdateUserName)
	protectedAuth.Get("/me/context", authController.GetMyContext)
	protectedAuth.Get("/me/simple-context", authController.GetMySimpleContext)
}
