package route

import (
	controller "masjidku_backend/internals/features/users/auth/controller"
	rateLimiter "masjidku_backend/internals/middlewares"
	authMw "masjidku_backend/internals/middlewares/auth"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AuthRoutes(app *fiber.App, db *gorm.DB) {
	authController := controller.NewAuthController(db)

	app.Use(rateLimiter.GlobalRateLimiter()) // ✅ masih aman di sini

	publicAuth := app.Group("/auth")

	publicAuth.Post("/login", rateLimiter.LoginRateLimiter(), authController.Login)
	publicAuth.Post("/register", rateLimiter.RegisterRateLimiter(), authController.Register)
	publicAuth.Post("/forgot-password/check", rateLimiter.ForgotPasswordRateLimiter(), authController.CheckSecurityAnswer)
	publicAuth.Post("/forgot-password/reset", authController.ResetPassword)
	publicAuth.Post("/login-google", authController.LoginGoogle)
	publicAuth.Post("/refresh-token", authController.RefreshToken)

	protectedAuth := app.Group("/api/auth", authMw.AuthMiddleware(db))
	protectedAuth.Post("/logout", authController.Logout)
	protectedAuth.Post("/change-password", authController.ChangePassword)
}
