// file: internals/features/school/class_schedules/routes/user_routes.go
package routes

import (
	dailyctl "masjidku_backend/internals/features/school/sessions/schedules/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// ScheduleUserRoutes mendaftarkan route untuk USER (read-only)
func ScheduleUserRoutes(user fiber.Router, db *gorm.DB) {
	sched := dailyctl.New(db, nil)

	// ✅ varian pakai masjid_id di path
	sg := user.Group("/:masjid_id/class-schedules")
	sg.Get("/list", sched.List)
	// Proyeksi jadwal → occurrences (kalender pengguna)
	// Query: ?from=YYYY-MM-DD&to=YYYY-MM-DD

}
