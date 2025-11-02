package route

import (
	"schoolku_backend/internals/features/schools/lecture_sessions/main/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Public read-only routes (tanpa login)
func AllLectureSessionRoutes(api fiber.Router, db *gorm.DB) {
	lectureSessionCtrl := controller.NewLectureSessionController(db)

	// 📚 Group: /lecture-sessions-u (public endpoints)
	pub := api.Group("/lecture-sessions-u")

	// by school id (publik jika controller mengizinkan)
	pub.Get("/by-school/:id", lectureSessionCtrl.GetLectureSessionsBySchoolIDParam)

	// by lecture id/slug
	pub.Get("/by-lecture/:lecture_id", lectureSessionCtrl.GetLectureSessionsByLectureID)
	pub.Get("/by-lecture-slug/:lecture_slug/all", lectureSessionCtrl.GetAllLectureSessionsByLectureSlug)

	// by session id/slug
	pub.Get("/by-id/:id", lectureSessionCtrl.GetLectureSessionByID)
	pub.Get("/by-slug/:slug", lectureSessionCtrl.GetLectureSessionBySlug)

	// grouping per bulan (berdasarkan school slug)
	pub.Get("/by-school-slug/:slug/group-by-month", lectureSessionCtrl.GetLectureSessionsGroupedByMonth)
	pub.Get("/by-school-slug/:slug/by-month/:month", lectureSessionCtrl.GetLectureSessionsByMonth)
	pub.Get("/by-school-slug/:slug", lectureSessionCtrl.GetLectureSessionsBySchoolID) // jika memang endpoint ini public-ready

	// list mendatang & selesai (publik)
	pub.Get("/mendatang/:slug", lectureSessionCtrl.GetUpcomingLectureSessionsBySchoolSlug)
	pub.Get("/soal-materi/:slug", lectureSessionCtrl.GetFinishedLectureSessionsBySchoolSlug)
}
