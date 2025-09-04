// file: internals/features/school/class_schedules/routes/user_routes.go
package routes

import (
	dailyctl "masjidku_backend/internals/features/school/sessions_assesment/schedule_daily/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// ScheduleUserRoutes mendaftarkan route untuk USER (read-only)
func ScheduleUserRoutes(user fiber.Router, db *gorm.DB) {
	// class_daily (occurrence harian)
	daily := dailyctl.NewClassDailyController(db)
	dg := user.Group("/class-daily")
	dg.Get("/", daily.List)
	dg.Get("/:id", daily.GetByID)

	// class_schedules (rencana)
	sched := dailyctl.New(db, nil) // validator nil
	sg := user.Group("/class-schedules")
	sg.Get("/", sched.List)
	sg.Get("/:id", sched.GetByID)
}

/*
Contoh mount di main.go:

  apiUser := app.Group("/api/u") // middleware auth user/public sesuai kebutuhan
  routes.ScheduleUserRoutes(apiUser, db)
*/
