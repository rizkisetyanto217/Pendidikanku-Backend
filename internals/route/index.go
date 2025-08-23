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

	// ===================== SHARED GROUPS =====================

	// PUBLIC: pakai SecondAuth biar bisa baca user_id kalau ada token (opsional)
	log.Println("[INFO] Setting up PUBLIC group...")
	public := app.Group("/public", authMiddleware.SecondAuthMiddleware(db))

	// PRIVATE (user login)
	log.Println("[INFO] Setting up PRIVATE group...")
	private := app.Group("/api/u", authMiddleware.AuthMiddleware(db))

	// ADMIN (login + must be admin/dkm/owner + scope masjid)
	log.Println("[INFO] Setting up ADMIN group (Auth + IsMasjidAdmin)...")
	admin := app.Group("/api/a",
		authMiddleware.AuthMiddleware(db),
		masjidkuMiddleware.IsMasjidAdmin(), 
	)

	// ===================== MASJID ROUTES =====================
	log.Println("[INFO] Mounting Masjid routes...")
	routeDetails.MasjidPublicRoutes(public, db)
	routeDetails.MasjidUserRoutes(private, db)
	routeDetails.MasjidAdminRoutes(admin, db) // ⬅️ semua admin masjid lewat group ini

	// ===================== LEMBAGA ROUTES =====================
	// Catatan: kalau rute lembaga juga butuh scope masjid, tetap pakai `admin` ini.
	// Kalau tidak butuh scope, buat group sendiri tanpa IsMasjidAdmin (di luar /api/a).
	log.Println("[INFO] Mounting Lembaga routes...")
	routeDetails.LembagaPublicRoutes(public, db)
	routeDetails.LembagaUserRoutes(private, db)
	routeDetails.LembagaAdminRoutes(admin, db)

	// ===================== HOME ROUTES =====================
	log.Println("[INFO] Mounting Home routes...")
	routeDetails.HomePublicRoutes(public, db)
	routeDetails.HomePrivateRoutes(private, db)
	routeDetails.HomeAdminRoutes(admin, db) // ⬅️ pakai group admin yang sama
}
