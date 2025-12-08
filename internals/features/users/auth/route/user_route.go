// file: internals/features/users/auth/route/auth_routes.go
package route

import (
	controller "madinahsalam_backend/internals/features/users/auth/controller"
	rateLimiter "madinahsalam_backend/internals/middlewares"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AuthRoutes(app *fiber.App, db *gorm.DB) {
	authController := controller.NewAuthController(db)

	// rate limiter global
	app.Use(rateLimiter.GlobalRateLimiter())

	// ==========================
	// GLOBAL AUTH (TANPA school_slug)
	// Base: /api/auth
	// ==========================
	baseAuth := app.Group("/api/auth")

	// CSRF & refresh tetap di sini (sesuai cookie path)
	baseAuth.Get("/csrf", authController.CSRF)
	baseAuth.Post("/refresh-token", authController.RefreshToken)

	// 🔓 Public global (owner / user biasa, belum tentu punya school)
	baseAuth.Post("/login", rateLimiter.LoginRateLimiter(), authController.Login)
	baseAuth.Post("/register", rateLimiter.RegisterRateLimiter(), authController.Register)
	baseAuth.Post("/forgot-password/reset", authController.ResetPassword)
	// Kalau nanti login-google mau global juga, bisa taruh di sini:
	// baseAuth.Post("/login-google", authController.LoginGoogle)

	// (Opsional, tapi enak punya versi global juga)
	baseAuth.Post("/logout", authController.Logout)
	baseAuth.Post("/change-password", authController.ChangePassword)
	baseAuth.Put("/update-user-name", authController.UpdateUserName)
	baseAuth.Get("/me/context", authController.GetMyContext)
	baseAuth.Get("/me/simple-context", authController.GetMySimpleContext)
	baseAuth.Get("/me/profile-completion", authController.GetMyProfileCompletion)

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
	// PROTECTED (SCOPED BY school_slug)
	// Base: /api/:school_slug/auth
	// ==========================
	protectedAuth := app.Group("/api/:school_slug/auth")

	protectedAuth.Post("/logout", authController.Logout)
	protectedAuth.Post("/change-password", authController.ChangePassword)
	protectedAuth.Put("/update-user-name", authController.UpdateUserName)
	protectedAuth.Get("/me/context", authController.GetMyContext)
	protectedAuth.Get("/me/simple-context", authController.GetMySimpleContext)
	protectedAuth.Get("/me/profile-completion", authController.GetMyProfileCompletion)
}
