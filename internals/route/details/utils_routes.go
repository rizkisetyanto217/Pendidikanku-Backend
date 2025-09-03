package details

import (
	tooltipRoute "masjidku_backend/internals/features/utils/tooltips/route"

	rateLimiter "masjidku_backend/internals/middlewares"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func UtilsRoutes(app *fiber.App, db *gorm.DB) {
	api := app.Group("/api",
		rateLimiter.GlobalRateLimiter(),
	)

	// 🔐 Untuk admin/teacher/owner
	adminGroup := api.Group("/a")
	tooltipRoute.TooltipAdminRoutes(adminGroup, db)

	// ✅ Route non-auth / publik
	publicGroup := app.Group("/api/n") // /n = no auth
	tooltipRoute.TooltipPublicRoutes(publicGroup, db)
}