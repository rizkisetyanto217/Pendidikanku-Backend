// file: internals/features/school/sessions_assesment/sessions/route/user_routes.go
package route

import (
	uaCtrl "masjidku_backend/internals/features/school/classes/class_attendance_sessions/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Contoh penggunaan middleware auth jika ada:
// import mw "masjidku_backend/internals/middlewares"

func AttendanceSessionsUserRoutes(r fiber.Router, db *gorm.DB) {
	// ✅ Group dengan masjid context
	masjidGroup := r.Group("/:masjid_id")

	// Attendance Sessions
	attendanceSessionController := uaCtrl.NewClassAttendanceSessionController(db)
	attendanceSessionGroup := masjidGroup.Group("/sessions" /* , mw.AuthRequired() */)
	attendanceSessionGroup.Get("/list", attendanceSessionController.ListClassAttendanceSessions) // GET /:masjid_id/sessions/list

	// User Attendance (read-only + CRUD untuk user)
	ua := uaCtrl.NewUserAttendanceController(db)
	uaGroup := masjidGroup.Group("/user-sessions")
	uaGroup.Get("/list", ua.List)
	uaGroup.Post("/", ua.CreateWithURLs) // POST /:masjid_id/user-sessions
	uaGroup.Patch("/:id", ua.Patch)      // PATCH /:masjid_id/user-sessions/:id
	uaGroup.Delete("/:id", ua.Delete)

	// User Attendance Types (read-only)
	uattCtl := uaCtrl.NewUserAttendanceTypeController(db)
	uatt := masjidGroup.Group("/user-attendance-types")
	uatt.Get("/", uattCtl.List)
}
