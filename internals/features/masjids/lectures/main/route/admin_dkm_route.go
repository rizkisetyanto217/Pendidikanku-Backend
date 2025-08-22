// file: internals/routes/lecture_admin_routes.go
package route

import (
	"masjidku_backend/internals/constants"
	"masjidku_backend/internals/features/masjids/lectures/main/controller"
	authMiddleware "masjidku_backend/internals/middlewares/auth"
	masjidkuMiddleware "masjidku_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func LectureAdminRoutes(api fiber.Router, db *gorm.DB) {
	// Guard global: wajib login + role admin/dkm/owner + scope masjid
	admin := api.Group("/",
		authMiddleware.AuthMiddleware(db),
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola lecture"),
			constants.AdminAndAbove, // admin, dkm, owner
		),
		masjidkuMiddleware.IsMasjidAdmin(), // inject masjid_id dari token
	)

	// 🔹 Lectures (CUD only)
	lectureCtrl := controller.NewLectureController(db)
	lecture := admin.Group("/lectures")
	lecture.Post("/", lectureCtrl.CreateLecture)
	lecture.Put("/:id", lectureCtrl.UpdateLecture)
	lecture.Delete("/:id", lectureCtrl.DeleteLecture)

	// 🔹 User Lectures (CUD admin) — mis. enroll paksa/administratif
	userLectureCtrl := controller.NewUserLectureController(db)
	userLecture := admin.Group("/user-lectures")
	userLecture.Post("/", userLectureCtrl.CreateUserLecture)          // admin assign
	userLecture.Post("/by-lecture", userLectureCtrl.GetUsersByLecture) // (ops) bisa tetap POST jika filter kompleks; kalau read-only pindah ke admin GET khusus

	// 🔹 Lecture Stats (CUD only)
	statsCtrl := controller.NewLectureStatsController(db)
	stats := admin.Group("/lecture-stats")
	stats.Post("/", statsCtrl.CreateLectureStats)
	stats.Put("/:lectureId", statsCtrl.UpdateLectureStats)

	// 🔹 Lecture Schedules (CUD only)
	lectureSchedulesCtrl := controller.NewLectureSchedulesController(db)
	schedule := admin.Group("/lecture-schedules")
	schedule.Post("/", lectureSchedulesCtrl.Create)
	schedule.Put("/:id", lectureSchedulesCtrl.Update)
	schedule.Delete("/:id", lectureSchedulesCtrl.Delete)
}
