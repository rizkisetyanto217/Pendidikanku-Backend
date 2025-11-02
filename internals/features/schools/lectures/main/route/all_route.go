// file: internals/routes/lecture_public_routes.go
package route

import (
	"schoolku_backend/internals/features/schools/lectures/main/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Mount di parent group: /api/v1/public (opsional pakai SecondAuthMiddleware di parent)
func AllLectureRoutes(api fiber.Router, db *gorm.DB) {
	// 🔹 Lectures (read-only)
	lectureCtrl := controller.NewLectureController(db)
	lecture := api.Group("/lectures")
	lecture.Get("/", lectureCtrl.GetAllLectures)
	lecture.Get("/:id/lecture-sessions", lectureCtrl.GetLectureSessionsByLectureID)
	lecture.Get("/:slug/lecture-sessions-by-slug", lectureCtrl.GetLectureSessionsByLectureSlug)
	lecture.Get("/:id", lectureCtrl.GetLectureByIDProgressUser) // safe read
	lecture.Get("/by-slug/:slug", lectureCtrl.GetLectureBySlugProgressUser)
	lecture.Get("/by-school-slug/:slug", lectureCtrl.GetLectureBySchoolSlug)

	// ❌ Hapus POST di public: CreateUserLecture & GetUsersByLecture dipindah ke /u (lihat bagian 3)

	// 🔹 Lecture Schedules (public read)
	lectureSchedulesCtrl := controller.NewLectureSchedulesController(db)
	api.Get("/lecture-schedules/by-school/:slug", lectureSchedulesCtrl.GetBySchoolSlug)
}
