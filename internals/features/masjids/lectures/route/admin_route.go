package route

import (
	"masjidku_backend/internals/features/masjids/lectures/controller"
	masjidkuMiddleware "masjidku_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func LectureRoutes(api fiber.Router, db *gorm.DB) {
	// 🔹 Lectures
	lectureCtrl := controller.NewLectureController(db)
	lecture := api.Group("/lectures")
	lecture.Post("/", masjidkuMiddleware.IsMasjidAdmin(), lectureCtrl.CreateLecture)
	lecture.Get("/by-masjid",masjidkuMiddleware.IsMasjidAdmin(), lectureCtrl.GetByMasjidID)
	lecture.Put("/:id", masjidkuMiddleware.IsMasjidAdmin(), lectureCtrl.UpdateLecture)
	lecture.Delete("/:id", masjidkuMiddleware.IsMasjidAdmin(), lectureCtrl.DeleteLecture)

	// 🔹 User Lectures
	userLectureCtrl := controller.NewUserLectureController(db)
	userLecture := api.Group("/user-lectures")
	userLecture.Post("/", masjidkuMiddleware.IsMasjidAdmin(), userLectureCtrl.CreateUserLecture)
	userLecture.Post("/by-lecture", masjidkuMiddleware.IsMasjidAdmin(), userLectureCtrl.GetUsersByLecture)

	// 🔹 Lecture Stats
	statsCtrl := controller.NewLectureStatsController(db)
	stats := api.Group("/lecture-stats")
	stats.Post("/", masjidkuMiddleware.IsMasjidAdmin(), statsCtrl.CreateLectureStats)
	stats.Get("/:lectureId", masjidkuMiddleware.IsMasjidAdmin(), statsCtrl.GetLectureStatsByLectureID)
	stats.Put("/:lectureId", masjidkuMiddleware.IsMasjidAdmin(), statsCtrl.UpdateLectureStats)
}
