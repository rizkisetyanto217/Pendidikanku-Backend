// file: internals/routes/setup.go
package routes

import (
	"log"
	"os"
	"strconv" // ⬅️ tambah ini
	"time"

	schoolkuMiddleware "madinahsalam_backend/internals/middlewares/auth_school"
	featuresMiddleware "madinahsalam_backend/internals/middlewares/features"

	routeDetails "madinahsalam_backend/internals/route/details"

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
	public := app.Group("/api/public")

	// ===================== PRIVATE (USER) =====================
	log.Println("[INFO] Setting up PRIVATE (loose) group...")
	privateLoose := app.Group("/api/u",
		schoolkuMiddleware.AuthJWT(schoolkuMiddleware.AuthJWTOpts{
			Secret:              os.Getenv("JWT_SECRET"),
			AllowCookieFallback: true,
		}),
	)

	log.Println("[INFO] Setting up PRIVATE (scoped) group...")
	privateScoped := app.Group("/api/u",
		schoolkuMiddleware.AuthJWT(schoolkuMiddleware.AuthJWTOpts{
			Secret:              os.Getenv("JWT_SECRET"),
			AllowCookieFallback: true,
		}),
	)

	// ===================== ADMIN (per school) =====================
	log.Println("[INFO] Setting up ADMIN group (Auth + Scope + RoleCheck)...")
	admin := app.Group("/api/a",
		schoolkuMiddleware.AuthJWT(schoolkuMiddleware.AuthJWTOpts{
			Secret:              os.Getenv("JWT_SECRET"),
			AllowCookieFallback: true,
		}),
		featuresMiddleware.UseSchoolScope(),
		featuresMiddleware.RequirePathScopeMatch(),
		featuresMiddleware.IsSchoolAdmin(),
	)

	// ===================== OWNER (GLOBAL) =====================
	log.Println("[INFO] Setting up OWNER group (Auth + owner global)...")
	owner := app.Group("/api/o",
		schoolkuMiddleware.AuthJWT(schoolkuMiddleware.AuthJWTOpts{
			Secret:              os.Getenv("JWT_SECRET"),
			AllowCookieFallback: true,
		}),
		featuresMiddleware.IsOwnerGlobal(),
	)

	// ===== Midtrans config (dipass ke FinanceAdminRoutes) =====
	midtransServerKey := os.Getenv("MIDTRANS_SERVER_KEY")
	useMidtransProd := func() bool {
		if v := os.Getenv("MIDTRANS_USE_PROD"); v != "" {
			b, err := strconv.ParseBool(v)
			if err == nil {
				return b
			}
		}
		return false
	}()

	// ===================== MOUNT ROUTES =====================

	log.Println("[INFO] Mounting Lembaga routes...")
	routeDetails.LembagaPublicRoutes(public, db)
	routeDetails.LembagaUserRoutes(privateScoped, db)
	routeDetails.LembagaAdminRoutes(admin, db)
	routeDetails.LembagaOwnerRoutes(owner, db)

	// 🔓 Mount route JOIN GLOBAL (tanpa school_id) KE privateLoose
	routeDetails.ClassSectionUserGlobalRoutes(privateLoose, db)

	log.Println("[INFO] Mounting Finance routes...")
	routeDetails.FinancePublicRoutes(public, db)
	// routeDetails.FinanceUserRoutes(privateScoped, db)
	routeDetails.FinanceAdminRoutes(admin, db, midtransServerKey, useMidtransProd) // ⬅️ FIX: pass 4 argumen
	routeDetails.FinanceOwnerRoutes(owner, db)
	routeDetails.FinanceUserRoutes(privateScoped, db)


	log.Println("[INFO] Mounting School routes...")
	routeDetails.SchoolPublicRoutes(public, db)
	routeDetails.SchoolUserRoutes(privateScoped, db)
	routeDetails.SchoolAdminRoutes(admin, db)
	routeDetails.SchoolOwnerRoutes(owner, db)
}
