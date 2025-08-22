// file: internals/routes/lecture_public_routes.go
package route

import (
	"masjidku_backend/internals/features/masjids/lectures/main/controller"

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
	lecture.Get("/:id", lectureCtrl.GetLectureByIDProgressUser)       // safe read
	lecture.Get("/by-slug/:slug", lectureCtrl.GetLectureBySlugProgressUser)
	lecture.Get("/by-masjid-slug/:slug", lectureCtrl.GetLectureByMasjidSlug)

	// ❌ Hapus POST di public: CreateUserLecture & GetUsersByLecture dipindah ke /u (lihat bagian 3)

	// 🔹 Lecture Schedules (public read)
	lectureSchedulesCtrl := controller.NewLectureSchedulesController(db)
	api.Get("/lecture-schedules/by-masjid/:slug", lectureSchedulesCtrl.GetByMasjidSlug)
}
