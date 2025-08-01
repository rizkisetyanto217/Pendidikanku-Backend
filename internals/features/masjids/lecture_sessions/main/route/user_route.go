package route

import (
	"masjidku_backend/internals/features/masjids/lecture_sessions/main/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AllLectureSessionRoutes(user fiber.Router, db *gorm.DB) {
	lectureSessionCtrl := controller.NewLectureSessionController(db)
	userLectureSessionCtrl := controller.NewUserLectureSessionController(db)

	// 📚 Group: /lecture-sessions
	session := user.Group("/lecture-sessions")
	session.Get("/", lectureSessionCtrl.GetAllLectureSessions)    // 📄 Lihat semua sesi

	session.Get("/by-masjid", lectureSessionCtrl.GetLectureSessionsByMasjidID)
	session.Get("/lecture-sessions", lectureSessionCtrl.GetByLectureID)
	
	
	// 👥 Group: /lecture-sessions-user
	sessionUser := user.Group("/lecture-sessions-u")
	sessionUser.Get("/by-masjid/:id", lectureSessionCtrl.GetLectureSessionsByMasjidIDParam)
	sessionUser.Get("/by-lecture/:lecture_id", lectureSessionCtrl.GetLectureSessionsByLectureID)
	sessionUser.Get("/by-id/:id", lectureSessionCtrl.GetLectureSessionByIDProgressUser)
	sessionUser.Get("/by-masjid-slug/:slug", lectureSessionCtrl.GetLectureSessionsByMasjidSlug)
	sessionUser.Get("/mendatang/:slug", lectureSessionCtrl.GetUpcomingLectureSessionsByMasjidSlug)
	sessionUser.Get("/soal-materi/:slug", lectureSessionCtrl.GetFinishedLectureSessionsByMasjidSlug)

	

	// 👥 Group: /user-lecture-sessions
	userSession := user.Group("/user-lecture-sessions")
	userSession.Post("/", userLectureSessionCtrl.CreateUserLectureSession) // ✅ Catat kehadiran / progress
	userSession.Get("/", userLectureSessionCtrl.GetAllUserLectureSessions)    // 🔍 Lihat semua sesi yang diikuti
	userSession.Get("/:id", userLectureSessionCtrl.GetUserLectureSessionByID) // 🔍 Detail kehadiran

}