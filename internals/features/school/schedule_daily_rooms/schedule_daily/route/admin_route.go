// file: internals/features/school/class_schedules/routes/admin_routes.go
package routes

import (
	dailyctl "masjidku_backend/internals/features/school/schedule_daily_rooms/schedule_daily/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// ScheduleAdminRoutes mendaftarkan route untuk ADMIN (CRUD penuh)
func ScheduleAdminRoutes(admin fiber.Router, db *gorm.DB) {
	// Controller untuk class_daily (occurrence)
	daily := dailyctl.NewClassDailyController(db)

	grpDaily := admin.Group("/class-daily")
	grpDaily.Get("/", daily.List)
	grpDaily.Get("/:id", daily.GetByID)
	grpDaily.Post("/", daily.Create)
	grpDaily.Put("/:id", daily.Update)
	grpDaily.Patch("/:id", daily.Patch)
	grpDaily.Delete("/:id", daily.Delete)

	// Controller untuk class_schedules (rencana)
	// Constructor jadwal masih menerima validator, kirim nil sesuai arahan "tanpa validator"
	sched := dailyctl.New(db, nil)

	grpSched := admin.Group("/class-schedules")
	grpSched.Get("/", sched.List)
	grpSched.Get("/:id", sched.GetByID)
	grpSched.Post("/", sched.Create)
	grpSched.Put("/:id", sched.Update)
	grpSched.Patch("/:id", sched.Patch)
	grpSched.Delete("/:id", sched.Delete)
}

/*
Contoh mount di main.go:

  apiAdmin := app.Group("/api/a") // sudah ada middleware auth + scope admin
  routes.ScheduleAdminRoutes(apiAdmin, db)
*/
