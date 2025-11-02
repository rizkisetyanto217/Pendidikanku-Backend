package route

import (
	"schoolku_backend/internals/constants"
	"schoolku_backend/internals/features/schools/lecture_sessions/main/controller"
	authMiddleware "schoolku_backend/internals/middlewares/auth"
	schoolkuMiddleware "schoolku_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Admin/DKM routes (CUD & approve)
func LectureSessionAdminRoutes(api fiber.Router, db *gorm.DB) {
	// Guard global: login + role admin/dkm/owner + scope school
	admin := api.Group("/",
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola sesi kajian"),
			constants.AdminAndAbove,
		),
		schoolkuMiddleware.IsSchoolAdmin(),
	)

	lectureSessionCtrl := controller.NewLectureSessionController(db)
	userLectureSessionCtrl := controller.NewUserLectureSessionController(db)

	// 📚 Group: /lecture-sessions
	ls := admin.Group("/lecture-sessions")
	ls.Post("/", lectureSessionCtrl.CreateLectureSession)
	ls.Get("/", lectureSessionCtrl.GetAllLectureSessions)
	ls.Get("/by-school", lectureSessionCtrl.GetLectureSessionsBySchoolID)
	ls.Get("/by-id/:id", lectureSessionCtrl.GetLectureSessionByID)

	ls.Put("/:id", lectureSessionCtrl.UpdateLectureSession)
	ls.Delete("/:id", lectureSessionCtrl.DeleteLectureSession)

	// ✅ Approve routes
	ls.Put("/:id/approve", lectureSessionCtrl.ApproveLectureSession)
	ls.Patch("/:id/approve-dkm", lectureSessionCtrl.ApproveLectureSessionByDKM)

	// 👥 Group: /user-lecture-sessions (admin view/manajemen data user)
	uls := admin.Group("/user-lecture-sessions")
	uls.Get("/", userLectureSessionCtrl.GetAllUserLectureSessions)
	uls.Get("/:id", userLectureSessionCtrl.GetUserLectureSessionByID)
	uls.Put("/:id", userLectureSessionCtrl.UpdateUserLectureSession)
	uls.Delete("/:id", userLectureSessionCtrl.DeleteUserLectureSession)
}
