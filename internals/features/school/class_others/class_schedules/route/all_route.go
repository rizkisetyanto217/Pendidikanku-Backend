// file: internals/features/school/class_schedules/routes/user_routes.go
package routes

import (
	nhctl "madinahsalam_backend/internals/features/school/class_others/class_schedules/controller"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// AllScheduleRoutes mendaftarkan route untuk USER / PUBLIC (read-only)
func AllScheduleRoutes(user fiber.Router, db *gorm.DB) {
	// Controller jadwal (header + optional rules)
	sched := nhctl.New(db, nil)
	rules := nhctl.NewClassScheduleRuleListController(db)

	// ============================
	// VARIAN PATH: BY :school_id
	// ============================
	// /api/u/:school_id/class-schedules/list
	// /api/u/:school_id/class-schedule-rules/list
	baseByID := user.Group("/:school_id")

	sgByID := baseByID.Group("/class-schedules")
	sgByID.Get("/list", sched.List)

	rgByID := baseByID.Group("/class-schedule-rules")
	rgByID.Get("/list", rules.ListPublic)

	// ============================
	// VARIAN PATH: BY :school_slug
	// ============================
	// /api/u/slug/:school_slug/class-schedules/list
	// /api/u/slug/:school_slug/class-schedule-rules/list
	baseBySlug := user.Group("/slug/:school_slug")

	sgBySlug := baseBySlug.Group("/class-schedules")
	sgBySlug.Get("/list", sched.List)

	rgBySlug := baseBySlug.Group("/class-schedule-rules")
	rgBySlug.Get("/list", rules.ListPublic)

	// ============================
	// VARIAN GENERIK (TOKEN / QUERY)
	// ============================
	// /api/u/class-schedules/list
	// /api/u/class-schedule-rules/list
	//
	// School akan di-resolve di controller lewat:
	// 1. Token (ResolveSchoolContext → EnsureSchoolAccessDKM / dsb)
	// 2. Kalau nggak ada, bisa pakai ?school_id / ?school_slug
	sg := user.Group("/class-schedules")
	sg.Get("/list", sched.List)

	rg := user.Group("/class-schedule-rules")
	rg.Get("/list", rules.ListPublic)

	// ============================
	// National Holidays (tetap)
	// ============================
	ctl := nhctl.NewNationHoliday(db, validator.New())
	grp := user.Group("/holidays/national")
	grp.Get("/", ctl.List) // ?q&is_active&is_recurring&date_from&date_to&sort&limit&offset
	grp.Get("/:id", ctl.GetByID)
}
