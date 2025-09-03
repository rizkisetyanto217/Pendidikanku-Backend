package routes

import (
	"log"
	"os"
	"time"

	masjidkuMiddleware "masjidku_backend/internals/middlewares/auth_masjid" // AuthJWT + MasjidContext
	featuresMiddleware "masjidku_backend/internals/middlewares/features"
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

	// ===================== GROUPS =====================

	// PUBLIC → AuthJWT opsional
	log.Println("[INFO] Setting up PUBLIC group...")
	public := app.Group("/public",
		masjidkuMiddleware.AuthJWT(masjidkuMiddleware.AuthJWTOpts{
			Secret:              os.Getenv("JWT_SECRET"),
			AllowCookieFallback: true,
		}),
	)

	// PRIVATE → user login biasa
	log.Println("[INFO] Setting up PRIVATE group...")
	private := app.Group("/api/u",
		masjidkuMiddleware.AuthJWT(masjidkuMiddleware.AuthJWTOpts{
			Secret:              os.Getenv("JWT_SECRET"),
			AllowCookieFallback: true,
		}),
	)

	// ADMIN → login + harus admin/dkm/owner + konteks masjid
	// ⚠️ Penting: IsMasjidAdmin DIPASANG SEKALI di sini (jangan di sub-routes).
	log.Println("[INFO] Setting up ADMIN group (Auth + MasjidContext + Scope + RoleCheck)...")
	admin := app.Group("/api/a",
		masjidkuMiddleware.AuthJWT(masjidkuMiddleware.AuthJWTOpts{
			Secret:              os.Getenv("JWT_SECRET"),
			AllowCookieFallback: true,
		}),
		masjidkuMiddleware.MasjidContext(masjidkuMiddleware.MasjidContextOpts{
			DB:      db,
			AppMode: masjidkuMiddleware.ModeDKM, // validasi role dkm/owner di level masjid
		}),
		featuresMiddleware.UseMasjidScope(), // ⟵ pilih active_masjid_id & active_role (auto-pick kalau 1)
		featuresMiddleware.IsMasjidAdmin(),  // ⟵ authorize
	)

	// ===================== MOUNT ROUTES =====================
	log.Println("[INFO] Mounting Masjid routes...")
	routeDetails.MasjidPublicRoutes(public, db)
	routeDetails.MasjidUserRoutes(private, db)
	routeDetails.MasjidAdminRoutes(admin, db) // gunakan router 'admin' yg sudah diproteksi

	log.Println("[INFO] Mounting Lembaga routes...")
	routeDetails.LembagaPublicRoutes(public, db)
	routeDetails.LembagaUserRoutes(private, db)
	routeDetails.LembagaAdminRoutes(admin, db)

	log.Println("[INFO] Mounting Home routes...")
	routeDetails.HomePublicRoutes(public, db)
	routeDetails.HomePrivateRoutes(private, db)
	routeDetails.HomeAdminRoutes(admin, db)
}
