package route

import (
	"masjidku_backend/internals/features/masjids/lectures/main/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AllLectureRoutes(api fiber.Router, db *gorm.DB) {
	// 🔹 Lecture
	lectureCtrl := controller.NewLectureController(db)
	lecture := api.Group("/lectures")
	lecture.Get("/", lectureCtrl.GetAllLectures)
	lecture.Get("/:id/lecture-sessions", lectureCtrl.GetLectureSessionsByLectureID)
	lecture.Get("/:id", lectureCtrl.GetLectureByIDProgressUser)
	lecture.Get("/by-slug/:slug", lectureCtrl.GetLectureBySlugProgressUser)
	lecture.Get("/by-masjid-slug/:slug", lectureCtrl.GetLectureByMasjidSlug)

	// 🔹 User Lecture
	ctrl := controller.NewUserLectureController(db)
	userLecture := api.Group("/user-lectures")
	userLecture.Post("/", ctrl.CreateUserLecture)
	userLecture.Post("/by-lecture", ctrl.GetUsersByLecture)

	// 🔹 Lecture Schedules
	lectureSchedulesCtrl := controller.NewLectureSchedulesController(db)
	api.Get("/lecture-schedules/by-masjid/:slug", lectureSchedulesCtrl.GetByMasjidSlug)
}
