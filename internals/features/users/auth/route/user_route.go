package route

import (
	controller "schoolku_backend/internals/features/users/auth/controller"
	rateLimiter "schoolku_backend/internals/middlewares"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AuthRoutes(app *fiber.App, db *gorm.DB) {
	authController := controller.NewAuthController(db)

	// rate limiter global
	app.Use(rateLimiter.GlobalRateLimiter())

	// ==========================
	// PUBLIC (SCOPED BY school_slug)
	// Base: /api/:school_slug/auth
	// ==========================
	publicAuth := app.Group("/api/:school_slug/auth")

	publicAuth.Post("/login", rateLimiter.LoginRateLimiter(), authController.Login)
	publicAuth.Post("/register", rateLimiter.RegisterRateLimiter(), authController.Register)
	publicAuth.Post("/forgot-password/reset", authController.ResetPassword)
	// publicAuth.Post("/login-google", authController.LoginGoogle) // kalau nanti diaktifin, juga ikut slug

	// ==========================
	// CSRF & REFRESH TOKEN (GLOBAL)
	// Tetap di /api/auth supaya cocok dengan cookie path
	// ==========================
	baseAuth := app.Group("/api/auth")
	baseAuth.Get("/csrf", authController.CSRF)
	baseAuth.Post("/refresh-token", authController.RefreshToken)

	// ==========================
	// PROTECTED (SCOPED BY school_slug)
	// Base: /api/:school_slug/auth
	// ==========================
	protectedAuth := app.Group("/api/:school_slug/auth")

	protectedAuth.Post("/logout", authController.Logout)
	protectedAuth.Post("/change-password", authController.ChangePassword)
	protectedAuth.Put("/update-user-name", authController.UpdateUserName)
	protectedAuth.Get("/me/context", authController.GetMyContext)
	protectedAuth.Get("/me/simple-context", authController.GetMySimpleContext)
	// di AuthRoutes, setelah protectedAuth := app.Group("/api/:school_slug/auth")
	protectedAuth.Get("/me/profile-completion", authController.GetMyProfileCompletion)

}
