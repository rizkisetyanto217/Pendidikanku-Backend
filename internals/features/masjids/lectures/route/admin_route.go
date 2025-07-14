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
	lecture.Post("/", masjidkuMiddleware.IsMasjidAdmin(db), lectureCtrl.CreateLecture)
	lecture.Get("/:id", masjidkuMiddleware.IsMasjidAdmin(db), lectureCtrl.GetLectureByID)
	lecture.Put("/:id", masjidkuMiddleware.IsMasjidAdmin(db), lectureCtrl.UpdateLecture)
	lecture.Delete("/:id", masjidkuMiddleware.IsMasjidAdmin(db), lectureCtrl.DeleteLecture)

	// 🔹 User Lectures
	userLectureCtrl := controller.NewUserLectureController(db)
	userLecture := api.Group("/user-lectures")
	userLecture.Post("/", masjidkuMiddleware.IsMasjidAdmin(db), userLectureCtrl.CreateUserLecture)
	userLecture.Post("/by-lecture", masjidkuMiddleware.IsMasjidAdmin(db), userLectureCtrl.GetUsersByLecture)

	// 🔹 Lecture Stats
	statsCtrl := controller.NewLectureStatsController(db)
	stats := api.Group("/lecture-stats")
	stats.Post("/", masjidkuMiddleware.IsMasjidAdmin(db), statsCtrl.CreateLectureStats)
	stats.Get("/:lectureId", masjidkuMiddleware.IsMasjidAdmin(db), statsCtrl.GetLectureStatsByLectureID)
	stats.Put("/:lectureId", masjidkuMiddleware.IsMasjidAdmin(db), statsCtrl.UpdateLectureStats)
}
