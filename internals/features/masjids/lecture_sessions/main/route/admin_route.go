package route

import (
	"masjidku_backend/internals/features/masjids/lecture_sessions/main/controller"
	masjidkuMiddleware "masjidku_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func LectureSessionAdminRoutes(admin fiber.Router, db *gorm.DB) {
	lectureSessionCtrl := controller.NewLectureSessionController(db)
	userLectureSessionCtrl := controller.NewUserLectureSessionController(db)

	// 📚 Group: /lecture-sessions
	admin.Post("/lecture-sessions", masjidkuMiddleware.IsMasjidAdmin(), lectureSessionCtrl.CreateLectureSession)
	admin.Get("/lecture-sessions", masjidkuMiddleware.IsMasjidAdmin(), lectureSessionCtrl.GetAllLectureSessions)
	admin.Get("/lecture-sessions/by-masjid", masjidkuMiddleware.IsMasjidAdmin(),lectureSessionCtrl.GetLectureSessionsByMasjidID)
	admin.Get("/lecture-sessions/by-id/:id", masjidkuMiddleware.IsMasjidAdmin(), lectureSessionCtrl.GetLectureSessionByID)

	admin.Put("/lecture-sessions/:id", masjidkuMiddleware.IsMasjidAdmin(), lectureSessionCtrl.UpdateLectureSession)
	admin.Delete("/lecture-sessions/:id", masjidkuMiddleware.IsMasjidAdmin(), lectureSessionCtrl.DeleteLectureSession)

	// ✅ Role-based approve (tanpa middleware IsMasjidAdmin)db
	admin.Put("/lecture-sessions/:id/approve", masjidkuMiddleware.IsMasjidAdmin(), lectureSessionCtrl.ApproveLectureSession)
	admin.Patch("/lecture-sessions/:id/approve-dkm", masjidkuMiddleware.IsMasjidAdmin(), lectureSessionCtrl.ApproveLectureSessionByDKM)


	// 👥 Group: /user-lecture-sessions
	admin.Get("/user-lecture-sessions", masjidkuMiddleware.IsMasjidAdmin(), userLectureSessionCtrl.GetAllUserLectureSessions)
	admin.Get("/user-lecture-sessions/:id", masjidkuMiddleware.IsMasjidAdmin(), userLectureSessionCtrl.GetUserLectureSessionByID)
	admin.Put("/user-lecture-sessions/:id", masjidkuMiddleware.IsMasjidAdmin(), userLectureSessionCtrl.UpdateUserLectureSession)
	admin.Delete("/user-lecture-sessions/:id", masjidkuMiddleware.IsMasjidAdmin(), userLectureSessionCtrl.DeleteUserLectureSession)
}
