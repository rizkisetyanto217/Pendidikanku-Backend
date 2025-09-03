package route

import (
	"masjidku_backend/internals/constants"
	"masjidku_backend/internals/features/masjids/lecture_sessions/main/controller"
	authMiddleware "masjidku_backend/internals/middlewares/auth"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// User routes (login wajib; untuk tindakan atas nama user sendiri)
func LectureSessionUserRoutes(api fiber.Router, db *gorm.DB) {
	r := api.Group("/",
		authMiddleware.OnlyRolesSlice(
			"❌ Hanya pengguna terautentikasi yang boleh mengakses fitur sesi kajian.",
			constants.AllowedRoles,
		),
	)

	lectureSessionCtrl := controller.NewLectureSessionController(db)
	userLectureSessionCtrl := controller.NewUserLectureSessionController(db)
	userAttendanceCtrl := controller.NewUserLectureSessionsAttendanceController(db)

	// (Opsional) read untuk user area—kalau ingin mirror sebagian GET
	usrRead := r.Group("/lecture-sessions")
	usrRead.Get("/", lectureSessionCtrl.GetAllLectureSessions)
	usrRead.Get("/by-masjid", lectureSessionCtrl.GetLectureSessionsByMasjidID)
	usrRead.Get("/by-lecture", lectureSessionCtrl.GetByLectureID) // sebelumnya "/lecture-sessions" → dibetulkan

	// 👥 Group: /user-lecture-sessions (aksi user)
	uls := r.Group("/user-lecture-sessions")
	uls.Post("/", userLectureSessionCtrl.CreateUserLectureSession)
	uls.Get("/", userLectureSessionCtrl.GetAllUserLectureSessions)
	uls.Get("/:id", userLectureSessionCtrl.GetUserLectureSessionByID)

	// 📝 Attendance & personal notes (login user)
	att := r.Group("/user-lecture-sessions-attendance")
	att.Post("/", userAttendanceCtrl.CreateOrUpdate)
	att.Get("/:lecture_session_id", userAttendanceCtrl.GetByLectureSession)
	att.Get("/:lecture_session_slug/by-slug", userAttendanceCtrl.GetByLectureSessionSlug)
	att.Delete("/:id", userAttendanceCtrl.Delete)
}
