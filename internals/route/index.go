// file: internals/routes/setup.go
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

	// ===================== AUTH / USER BASE =====================
	log.Println("[INFO] Setting up AuthRoutes...")
	routeDetails.AuthRoutes(app, db)

	log.Println("[INFO] Setting up UserRoutes...")
	routeDetails.UserRoutes(app, db)

	// ===================== GROUPS =====================

	// PUBLIC → JWT opsional
	log.Println("[INFO] Setting up PUBLIC group...")
	public := app.Group("/public")

	// ===================== PRIVATE (USER) =====================
	// 🔓 privateLoose: TANPA scope/role strict. Dipakai untuk endpoint yang
	//     tidak membutuhkan masjid_id pada path (contoh: join by code).
	log.Println("[INFO] Setting up PRIVATE (loose) group...")
	privateLoose := app.Group("/api/u",
		masjidkuMiddleware.AuthJWT(masjidkuMiddleware.AuthJWTOpts{
			Secret:              os.Getenv("JWT_SECRET"),
			AllowCookieFallback: true,
		}),
	)

	// 🔒 privateScoped: (jika diperlukan) pasang middleware features di
	//     sub-paket yang memang butuh masjid scope. Di sini kita tidak
	//     memaksa UseMasjidScope global agar tidak menular ke endpoint loose.
	log.Println("[INFO] Setting up PRIVATE (scoped) group...")
	privateScoped := app.Group("/api/u",
		masjidkuMiddleware.AuthJWT(masjidkuMiddleware.AuthJWTOpts{
			Secret:              os.Getenv("JWT_SECRET"),
			AllowCookieFallback: true,
		}),
		// NOTE: JANGAN taruh UseMasjidScope di sini secara global
		// Jika sebuah paket user memang butuh scope strict,
		// pasang di file route paket tersebut (di level subgroup).
	)

	// ===================== ADMIN (per masjid) =====================
	log.Println("[INFO] Setting up ADMIN group (Auth + Scope + RoleCheck)...")
	admin := app.Group("/api/a",
		masjidkuMiddleware.AuthJWT(masjidkuMiddleware.AuthJWTOpts{
			Secret:              os.Getenv("JWT_SECRET"),
			AllowCookieFallback: true,
		}),
		featuresMiddleware.UseMasjidScope(),
		featuresMiddleware.RequirePathScopeMatch(),
		featuresMiddleware.IsMasjidAdmin(),
	)

	// ===================== OWNER (GLOBAL) =====================
	log.Println("[INFO] Setting up OWNER group (Auth + owner global)...")
	owner := app.Group("/api/o",
		masjidkuMiddleware.AuthJWT(masjidkuMiddleware.AuthJWTOpts{
			Secret:              os.Getenv("JWT_SECRET"),
			AllowCookieFallback: true,
		}),
		featuresMiddleware.IsOwnerGlobal(),
	)

	// ===================== MOUNT ROUTES =====================
	log.Println("[INFO] Mounting Masjid routes...")
	routeDetails.MasjidPublicRoutes(public, db)
	routeDetails.MasjidUserRoutes(privateScoped, db) // user routes lain → scoped (kalau perlu scope pasang di sub-group paketnya)
	routeDetails.MasjidAdminRoutes(admin, db)
	routeDetails.MasjidOwnerRoutes(owner, db)

	log.Println("[INFO] Mounting Lembaga routes...")
	routeDetails.LembagaPublicRoutes(public, db)
	routeDetails.LembagaUserRoutes(privateScoped, db) // biarkan paket ini pasang middleware scope di level subgroup-nya bila butuh
	routeDetails.LembagaAdminRoutes(admin, db)
	routeDetails.LembagaOwnerRoutes(owner, db)

	// 🔓 Mount route JOIN GLOBAL (tanpa masjid_id) KE privateLoose
	routeDetails.ClassSectionUserGlobalRoutes(privateLoose, db)

	log.Println("[INFO] Mounting Home routes...")
	routeDetails.HomePublicRoutes(public, db)
	routeDetails.HomePrivateRoutes(privateScoped, db)
	routeDetails.HomeAdminRoutes(admin, db)

	log.Println("[INFO] Mounting Finance routes...")
	routeDetails.FinancePublicRoutes(public, db)
	// routeDetails.FinanceUserRoutes(privateScoped, db)
	routeDetails.FinanceAdminRoutes(admin, db)
	routeDetails.FinanceOwnerRoutes(owner, db)

}
