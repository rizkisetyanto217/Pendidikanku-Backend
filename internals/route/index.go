// file: internals/routes/setup.go (atau file yang berisi SetupRoutes)
package routes

import (
	"log"
	"os"
	"time"

	masjidkuMiddleware "masjidku_backend/internals/middlewares/auth_masjid"
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

	// ADMIN (per-masjid) → butuh konteks masjid
	log.Println("[INFO] Setting up ADMIN group (Auth + MasjidContext + Scope + RoleCheck)...")
	admin := app.Group("/api/a",
		masjidkuMiddleware.AuthJWT(masjidkuMiddleware.AuthJWTOpts{
			Secret:              os.Getenv("JWT_SECRET"),
			AllowCookieFallback: true,
		}),
		masjidkuMiddleware.MasjidContext(masjidkuMiddleware.MasjidContextOpts{
			DB:      db,
			AppMode: masjidkuMiddleware.ModeDKM,
		}),
		featuresMiddleware.UseMasjidScope(),
		featuresMiddleware.IsMasjidAdmin(),
	)

	// ✅ OWNER (GLOBAL) → TANPA MasjidContext
	log.Println("[INFO] Setting up OWNER group (Auth + owner global)...")
	owner := app.Group("/api/o",
		masjidkuMiddleware.AuthJWT(masjidkuMiddleware.AuthJWTOpts{
			Secret:              os.Getenv("JWT_SECRET"),
			AllowCookieFallback: true,
		}),
		featuresMiddleware.IsOwnerGlobal(), // middleware baru di bawah
	)

	// ===================== MOUNT ROUTES =====================
	log.Println("[INFO] Mounting Masjid routes...")
	routeDetails.MasjidPublicRoutes(public, db)
	routeDetails.MasjidUserRoutes(private, db)
	routeDetails.MasjidAdminRoutes(admin, db) // per-masjid
	routeDetails.MasjidOwnerRoutes(owner, db)


	log.Println("[INFO] Mounting Lembaga routes...")
	routeDetails.LembagaPublicRoutes(public, db)
	routeDetails.LembagaUserRoutes(private, db)
	routeDetails.LembagaAdminRoutes(admin, db)
	routeDetails.LembagaOwnerRoutes(owner, db)

	log.Println("[INFO] Mounting Home routes...")
	routeDetails.HomePublicRoutes(public, db)
	routeDetails.HomePrivateRoutes(private, db)
	routeDetails.HomeAdminRoutes(admin, db)
}
