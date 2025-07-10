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

	// ===================== GENERAL FEATURES =====================
	log.Println("[INFO] Setting up UtilsRoutes...")
	routeDetails.UtilsRoutes(app, db)

	log.Println("[INFO] Setting up CertificateRoutes...")
	routeDetails.CertificateRoutes(app, db)

	// ===================== MASJID PAGE =====================
	log.Println("[INFO] Setting up MasjidRoutes (public)...")
	masjidPublic := app.Group("/public", authMiddleware.OptionalJWTMiddleware(db))
	routeDetails.MasjidPublicRoutes(masjidPublic, db)

	log.Println("[INFO] Setting up MasjidRoutes (private)...")
	masjidPrivate := app.Group("/api/u", authMiddleware.AuthMiddleware(db))
	routeDetails.MasjidUserRoutes(masjidPrivate, db)

	log.Println("[INFO] Setting up MasjidRoutes (admin)...")
	masjidAdmin := app.Group("/api/a",
		authMiddleware.AuthMiddleware(db),
		masjidkuMiddleware.IsMasjidAdmin(),
	)
	routeDetails.MasjidAdminRoutes(masjidAdmin, db)

	// ===================== HOME GROUPS =====================

	// ✅ 1. PUBLIC: Tidak butuh token
	log.Println("[INFO] Setting up HomeRoutes (public)...")
	public := app.Group("/public") // <= tidak ada auth
	routeDetails.HomePublicRoutes(public, db)

	// ✅ 2. PRIVATE: User biasa (wajib token)
	log.Println("[INFO] Setting up HomeRoutes (private)...")
	private := app.Group("/api", authMiddleware.AuthMiddleware(db))
	routeDetails.HomePrivateRoutes(private, db)

	// ✅ 3. ADMIN: Admin masjid (wajib token + admin check)
	log.Println("[INFO] Setting up HomeRoutes (admin)...")
	admin := app.Group("/api/a",
		authMiddleware.AuthMiddleware(db),
		masjidkuMiddleware.IsMasjidAdmin(),
	)
	routeDetails.HomeAdminRoutes(admin, db)
}
