// file: internals/features/school/class_schedules/routes/user_routes.go
package routes

import (
	nhctl "schoolku_backend/internals/features/school/classes/class_schedules/controller"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// ScheduleUserRoutes mendaftarkan route untuk USER / PUBLIC (read-only)
func ScheduleUserRoutes(user fiber.Router, db *gorm.DB) {
	// Controller jadwal (header + optional rules)
	sched := nhctl.New(db, nil)
	rules := nhctl.NewClassScheduleRuleListController(db)

	// ============================
	// JADWAL KELAS (TOKEN-BASED SCHOOL)
	// ============================
	//
	// Path:
	//   - GET /api/u/class-schedules/list
	//   - GET /api/u/class-schedule-rules/list
	//
	// school_id wajib di-resolve dari token di controller
	// (tidak ada lagi varian :school_id, :school_slug, atau ?school_id / ?school_slug)
	sg := user.Group("/class-schedules")
	sg.Get("/list", sched.List)

	rg := user.Group("/class-schedule-rules")
	rg.Get("/list", rules.ListPublic)

	// ============================
	// National Holidays (tetap tidak terkait token)
	// ============================
	ctl := nhctl.NewNationHoliday(db, validator.New())
	grp := user.Group("/holidays/national")
	grp.Get("/", ctl.List) // ?q&is_active&is_recurring&date_from&date_to&sort&limit&offset
	grp.Get("/:id", ctl.GetByID)
}
