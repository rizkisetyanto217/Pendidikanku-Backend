package route

import (
	controller "masjidku_backend/internals/features/users/auth/controller"
	rateLimiter "masjidku_backend/internals/middlewares"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AuthRoutes(app *fiber.App, db *gorm.DB) {
	authController := controller.NewAuthController(db)

	app.Use(rateLimiter.GlobalRateLimiter()) // ✅ masih aman di sini

	publicAuth := app.Group("/auth")

	publicAuth.Post("/login", rateLimiter.LoginRateLimiter(), authController.Login)
	publicAuth.Post("/register", rateLimiter.RegisterRateLimiter(), authController.Register)
	publicAuth.Post("/forgot-password/reset", authController.ResetPassword)
	publicAuth.Post("/login-google", authController.LoginGoogle)
	publicAuth.Post("/refresh-token", authController.RefreshToken)

	protectedAuth := app.Group("/api/auth")
	protectedAuth.Post("/logout", authController.Logout)
	protectedAuth.Post("/change-password", authController.ChangePassword)
	// ⬇️ Tambahkan ini:
	protectedAuth.Get("/me", authController.Me)
	protectedAuth.Put("/update-user-name", authController.UpdateUserName)
}
