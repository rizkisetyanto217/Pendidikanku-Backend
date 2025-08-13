package routes

import (
	"log"
	"time"

	authMiddleware "masjidku_backend/internals/middlewares/auth"
	masjidkuMiddleware "masjidku_backend/internals/middlewares/features"
	routeDetails "masjidku_backend/internals/route/details"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var startTime time.Time

func SetupRoutes(app *fiber.App, db *gorm.DB) {
	startTime = time.Now()

	// ===================== BASE =====================
	BaseRoutes(app, db)

	// ===================== AUTH / USER =====================
	log.Println("[INFO] Setting up AuthRoutes...")
	routeDetails.AuthRoutes(app, db)

	log.Println("[INFO] Setting up UserRoutes...")
	routeDetails.UserRoutes(app, db)

	log.Println("[INFO] Setting up UtilsRoutes...")
	routeDetails.UtilsRoutes(app, db)

	// ===================== MASJID PAGE =====================
	log.Println("[INFO] Setting up MasjidRoutes (public)...")
	masjidPublic := app.Group("/public", authMiddleware.SecondAuthMiddleware(db))
	routeDetails.MasjidPublicRoutes(masjidPublic, db)

	log.Println("[INFO] Setting up MasjidRoutes (private)...")
	masjidPrivate := app.Group("/api/u", authMiddleware.AuthMiddleware(db))
	routeDetails.MasjidUserRoutes(masjidPrivate, db)

	log.Println("[INFO] Setting up MasjidRoutes (admin)...")
	masjidAdmin := app.Group("/api/a", authMiddleware.AuthMiddleware(db))
	routeDetails.MasjidAdminRoutes(masjidAdmin, db)

	// ===================== LEMBAGA PAGE =====================
	log.Println("[INFO] Setting up LembagaRoutes (public)...")
	lembagaPublic := app.Group("/public", authMiddleware.SecondAuthMiddleware(db))
	routeDetails.LembagaPublicRoutes(lembagaPublic, db)

	log.Println("[INFO] Setting up LembagaRoutes (private)...")
	lembagaPrivate := app.Group("/api/u", authMiddleware.AuthMiddleware(db))
	routeDetails.LembagaUserRoutes(lembagaPrivate, db)

	log.Println("[INFO] Setting up LembagaRoutes (admin)...")
	lembagaAdmin := app.Group("/api/a", authMiddleware.AuthMiddleware(db))
	routeDetails.LembagaAdminRoutes(lembagaAdmin, db)

	// ===================== HOME GROUPS =====================

	// ✅ 1. PUBLIC: Gunakan SecondAuthMiddleware agar bisa dapat user_id jika login
	log.Println("[INFO] Setting up HomeRoutes (public)...")
	public := app.Group("/public", authMiddleware.SecondAuthMiddleware(db))
	routeDetails.HomePublicRoutes(public, db)

	// ✅ 2. PRIVATE: Wajib token
	log.Println("[INFO] Setting up HomeRoutes (private)...")
	private := app.Group("/api/u", authMiddleware.AuthMiddleware(db))
	routeDetails.HomePrivateRoutes(private, db)

	// ✅ 3. ADMIN: Wajib token + admin
	log.Println("[INFO] Setting up HomeRoutes (admin)...")
	admin := app.Group("/api/a",
		authMiddleware.AuthMiddleware(db),
		masjidkuMiddleware.IsMasjidAdmin(),
	)
	routeDetails.HomeAdminRoutes(admin, db)
}
