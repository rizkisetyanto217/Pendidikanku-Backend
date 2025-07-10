package route

import (
	"masjidku_backend/internals/features/masjids/lectures/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func LectureRoutes(api fiber.Router, db *gorm.DB) {
	// 🔹 Lectures
	lectureCtrl := controller.NewLectureController(db)
	lecture := api.Group("/lectures")
	lecture.Post("/", lectureCtrl.CreateLecture)

	// 🔹 User Lectures
	userLectureCtrl := controller.NewUserLectureController(db)
	userLecture := api.Group("/user-lectures")
	userLecture.Post("/", userLectureCtrl.CreateUserLecture)
	userLecture.Post("/by-lecture", userLectureCtrl.GetUsersByLecture)

	// 🔹 Lecture Stats
	statsCtrl := controller.NewLectureStatsController(db)
	stats := api.Group("/lecture-stats")
	stats.Post("/", statsCtrl.CreateLectureStats)
	stats.Get("/:lectureId", statsCtrl.GetLectureStatsByLectureID)
	stats.Put("/:lectureId", statsCtrl.UpdateLectureStats)
}
