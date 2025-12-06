// file: internals/features/school/class_schedules/routes/admin_routes.go
package routes

import (
	scheduleController "madinahsalam_backend/internals/features/school/class_others/class_schedules/controller/schedule"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// ScheduleAdminRoutes mendaftarkan route untuk ADMIN (CRUD penuh + sinkronisasi CAS)
func ScheduleAdminRoutes(admin fiber.Router, db *gorm.DB) {
	// constructor controller (validator nil sesuai arsitektur sekarang)
	sched := scheduleController.NewSchedule(db, nil)

	// ⬇️ tambahkan :school_id di path supaya helper ResolveSchoolContext bisa resolve dari path
	grpSched := admin.Group("/class-schedules")

	grpSched.Post("/", sched.Create)
	grpSched.Patch("/:id", sched.Patch)
	grpSched.Delete("/:id", sched.Delete)

}
