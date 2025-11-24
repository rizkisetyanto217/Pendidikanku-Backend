// internals/routes/semester_stats_routes.go
package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	statsCtl "madinahsalam_backend/internals/features/lembaga/stats/semester_stats/controller"
)

// Admin endpoints: GET all (with filters)
func UserClassAttendanceSemesterAdminRoutes(r fiber.Router, db *gorm.DB) {
	apiAdmin := r.Group("/api/a") // boleh r = app atau group lain

	ctl := statsCtl.NewSemesterStatsController(db)

	apiAdmin.Get("/semester-stats", ctl.List)
}
