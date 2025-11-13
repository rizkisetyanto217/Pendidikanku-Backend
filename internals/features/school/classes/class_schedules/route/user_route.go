// file: internals/features/school/class_schedules/routes/user_routes.go
package routes

import (
	nhctl "schoolku_backend/internals/features/school/classes/class_schedules/controller"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// ScheduleUserRoutes mendaftarkan route untuk USER (read-only)
func ScheduleUserRoutes(user fiber.Router, db *gorm.DB) {
	// Controller jadwal (header)
	sched := nhctl.New(db, nil)

	// ✅ varian pakai school_id di path
	sg := user.Group("/:school_id/class-schedules")
	sg.Get("/list", sched.List) // ?from=YYYY-MM-DD&to=YYYY-MM-DD

	// Proyeksi jadwal → occurrences (kalender pengguna)
	// Query: ?from=YYYY-MM-DD&to=YYYY-MM-DD

	// National holidays
	ctl := nhctl.NewNationHoliday(db, validator.New())
	grp := user.Group("/holidays/national")
	grp.Get("/", ctl.List) // ?q&is_active&is_recurring&date_from&date_to&sort&limit&offset
	grp.Get("/:id", ctl.GetByID)

	// ===============================
	// Class Schedule Rules (user list)
	// ===============================
	rules := nhctl.NewClassScheduleRuleListController(db)
	rg := user.Group("/:school_id/class-schedule-rules")
	rg.Get("/list", rules.List)
	// Query (opsional):
	//   ?schedule_id&dow&parity=all|odd|even
	//   &teacher_id&section_id&class_subject_id&room_id
	//   &sort_by=day_of_week|start_time|end_time|created_at|updated_at
	//   &order=asc|desc&limit&offset
}
