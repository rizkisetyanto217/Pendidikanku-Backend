// internals/routes/semester_stats_routes.go
package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	statsCtl "masjidku_backend/internals/features/lembaga/stats/semester_stats/controller"
)

func UserClassAttendanceSemesterUserRoutes(r fiber.Router, db *gorm.DB) {
	apiAdmin := r.Group("/api/a") // bisa r = app atau group lain

	ctl := statsCtl.NewSemesterStatsController(db)

	apiAdmin.Get("/semester-stats/by-user/:user_id", ctl.ListByUserID)
}
