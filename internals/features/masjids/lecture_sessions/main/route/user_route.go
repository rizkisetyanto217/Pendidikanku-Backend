package route

import (
	"masjidku_backend/internals/features/masjids/lecture_sessions/main/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AllLectureSessionRoutes(user fiber.Router, db *gorm.DB) {
	lectureSessionCtrl := controller.NewLectureSessionController(db)
	userLectureSessionCtrl := controller.NewUserLectureSessionController(db)
	userAttendanceCtrl := controller.NewUserLectureSessionsAttendanceController(db) // ✅ Tambah controller baru

	// 📚 Group: /lecture-sessions
	session := user.Group("/lecture-sessions")
	session.Get("/", lectureSessionCtrl.GetAllLectureSessions)
	session.Get("/by-masjid", lectureSessionCtrl.GetLectureSessionsByMasjidID)
	session.Get("/lecture-sessions", lectureSessionCtrl.GetByLectureID)

	// 👥 Group: /lecture-sessions-u
	sessionUser := user.Group("/lecture-sessions-u")
	sessionUser.Get("/by-masjid/:id", lectureSessionCtrl.GetLectureSessionsByMasjidIDParam)
	sessionUser.Get("/by-lecture/:lecture_id", lectureSessionCtrl.GetLectureSessionsByLectureID)
	sessionUser.Get("/by-id/:id", lectureSessionCtrl.GetLectureSessionByID)
	sessionUser.Get("/by-slug/:slug", lectureSessionCtrl.GetLectureSessionBySlug)	
	sessionUser.Get("/by-masjid-slug/:slug", lectureSessionCtrl.GetLectureSessionBySlug)
	sessionUser.Get("/mendatang/:slug", lectureSessionCtrl.GetUpcomingLectureSessionsByMasjidSlug)
	sessionUser.Get("/soal-materi/:slug", lectureSessionCtrl.GetFinishedLectureSessionsByMasjidSlug)

	// 👥 Group: /user-lecture-sessions
	userSession := user.Group("/user-lecture-sessions")
	userSession.Post("/", userLectureSessionCtrl.CreateUserLectureSession)
	userSession.Get("/", userLectureSessionCtrl.GetAllUserLectureSessions)
	userSession.Get("/:id", userLectureSessionCtrl.GetUserLectureSessionByID)

	// ✅ Tambah route untuk /user-lecture-sessions-attendance
	userAttendance := user.Group("/user-lecture-sessions-attendance")
	userAttendance.Post("/", userAttendanceCtrl.CreateOrUpdate)
	userAttendance.Get("/:lecture_session_id", userAttendanceCtrl.GetByLectureSession)
	userAttendance.Get("/:lecture_session_slug/by-slug", userAttendanceCtrl.GetByLectureSessionSlug)
	userAttendance.Delete("/:id", userAttendanceCtrl.Delete)
}
