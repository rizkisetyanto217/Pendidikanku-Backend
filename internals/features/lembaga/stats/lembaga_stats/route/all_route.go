// internals/features/lembaga/stats/route/lembaga_stats_route.go
package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"madinahsalam_backend/internals/features/lembaga/stats/lembaga_stats/controller"
)

func AllLembagaStatsRoutes(router fiber.Router, db *gorm.DB) {
	h := controller.NewLembagaStatsController(db)

	// Group untuk tenant (ambil school_id dari token)
	tenant := router.Group("/lembaga-stats")
	{
		tenant.Get("/:school_id", h.GetBySchoolID)
	}
}
