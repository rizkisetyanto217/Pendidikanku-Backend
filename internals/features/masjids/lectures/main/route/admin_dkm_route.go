// file: internals/routes/lecture_admin_routes.go
package route

import (
	"masjidku_backend/internals/features/masjids/lectures/main/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func LectureAdminRoutes(admin fiber.Router, db *gorm.DB) {
	lectureCtrl := controller.NewLectureController(db)
	userLectureCtrl := controller.NewUserLectureController(db)
	statsCtrl := controller.NewLectureStatsController(db)
	lectureSchedulesCtrl := controller.NewLectureSchedulesController(db)

	// Lectures
	admin.Post("/lectures",        lectureCtrl.CreateLecture)
	admin.Put("/lectures/:id",     lectureCtrl.UpdateLecture)
	admin.Delete("/lectures/:id",  lectureCtrl.DeleteLecture)

	// User Lectures
	admin.Post("/user-lectures",            userLectureCtrl.CreateUserLecture)
	admin.Post("/user-lectures/by-lecture", userLectureCtrl.GetUsersByLecture)

	// Lecture Stats
	admin.Post("/lecture-stats",              statsCtrl.CreateLectureStats)
	admin.Put("/lecture-stats/:lectureId",    statsCtrl.UpdateLectureStats)

	// Lecture Schedules
	admin.Post("/lecture-schedules",       lectureSchedulesCtrl.Create)
	admin.Put("/lecture-schedules/:id",    lectureSchedulesCtrl.Update)
	admin.Delete("/lecture-schedules/:id", lectureSchedulesCtrl.Delete)
}