// file: internals/features/school/class_schedules/routes/admin_routes.go
package routes

import (
	dailyctl "masjidku_backend/internals/features/school/sessions/schedules/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// ScheduleAdminRoutes mendaftarkan route untuk ADMIN (CRUD penuh + sinkronisasi CAS)
func ScheduleAdminRoutes(admin fiber.Router, db *gorm.DB) {
	// constructor controller (validator nil sesuai arsitektur sekarang)
	sched := dailyctl.New(db, nil)

	// ⬇️ tambahkan :masjid_id di path supaya helper ResolveMasjidContext bisa resolve dari path
	grpSched := admin.Group("/:masjid_id/class-schedules")

	grpSched.Post("/", sched.Create)
	grpSched.Patch("/:id", sched.Patch)
	grpSched.Delete("/:id", sched.Delete)

}
