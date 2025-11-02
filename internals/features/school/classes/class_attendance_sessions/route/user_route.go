// file: internals/features/school/sessions_assesment/sessions/route/user_routes.go
package route

import (
	uaCtrl "schoolku_backend/internals/features/school/classes/class_attendance_sessions/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Contoh penggunaan middleware auth jika ada:
// import mw "schoolku_backend/internals/middlewares"

func AttendanceSessionsUserRoutes(r fiber.Router, db *gorm.DB) {
	// ✅ Group dengan school context
	schoolGroup := r.Group("/:school_id")

	// Attendance Sessions
	attendanceSessionController := uaCtrl.NewClassAttendanceSessionController(db)
	attendanceSessionGroup := schoolGroup.Group("/sessions" /* , mw.AuthRequired() */)
	attendanceSessionGroup.Get("/list", attendanceSessionController.ListClassAttendanceSessions) // GET /:school_id/sessions/list

	// User Attendance (read-only + CRUD untuk user)
	ua := uaCtrl.NewStudentAttendanceController(db)
	uaGroup := schoolGroup.Group("/student-sessions")
	uaGroup.Get("/list", ua.List)
	uaGroup.Post("/", ua.CreateWithURLs) // POST /:school_id/user-sessions
	uaGroup.Patch("/:id", ua.Patch)      // PATCH /:school_id/user-sessions/:id
	uaGroup.Delete("/:id", ua.Delete)

	// User Attendance Types (read-only)
	uattCtl := uaCtrl.NewStudentAttendanceTypeController(db)
	uatt := schoolGroup.Group("/student-attendance-types")
	uatt.Get("/", uattCtl.List)
}
