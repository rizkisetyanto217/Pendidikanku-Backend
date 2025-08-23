// internals/features/lembaga/stats/route/lembaga_stats_route.go
package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"masjidku_backend/internals/features/lembaga/stats/lembaga_stats/controller"
)

func LembagaStatsAdminRoutes(router fiber.Router, db *gorm.DB) {
		h := controller.NewLembagaStatsController(db)

	// Group untuk tenant (ambil masjid_id dari token)
	tenant := router.Group("/lembaga-stats")
	{
		tenant.Get("/", h.GetMyLembagaStats)
		tenant.Post("/", h.CreateMyLembagaStats)
		tenant.Put("/", h.UpdateMyLembagaStats)
	}
}
