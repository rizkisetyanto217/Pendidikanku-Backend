// file: internals/features/school/class_schedules/routes/user_routes.go
package routes

import (
	nhctl "masjidku_backend/internals/features/school/sessions/schedules/controller"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// ScheduleUserRoutes mendaftarkan route untuk USER (read-only)
func ScheduleUserRoutes(user fiber.Router, db *gorm.DB) {
	sched := nhctl.New(db, nil)

	// ✅ varian pakai masjid_id di path
	sg := user.Group("/:masjid_id/class-schedules")
	sg.Get("/list", sched.List)
	// Proyeksi jadwal → occurrences (kalender pengguna)
	// Query: ?from=YYYY-MM-DD&to=YYYY-MM-DD

	ctl := nhctl.NewNationHoliday(db, validator.New())

	grp := user.Group("/holidays/national")

	grp.Get("/", ctl.List) // ?q&is_active&is_recurring&date_from&date_to&sort&limit&offset
	grp.Get("/:id", ctl.GetByID)
}
