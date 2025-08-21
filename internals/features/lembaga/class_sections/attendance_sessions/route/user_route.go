// internals/features/lembaga/class_sections/attendance_sessions/main/router/class_attendance_session_routes.go
package route

import (
	controller "masjidku_backend/internals/features/lembaga/class_sections/attendance_sessions/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Contoh penggunaan middleware auth jika ada:
// import mw "masjidku_backend/internals/middlewares"

func AttendanceSessionsUserRoutes(r fiber.Router, db *gorm.DB) {
	// Buat instance controller dengan nama jelas
	attendanceSessionController := controller.NewClassAttendanceSessionController(db)

	// Kelompokkan route di prefix yang jelas
	attendanceSessionGroup := r.Group("/class-attendance-sessions" /* , mw.AuthRequired() */)

	// List & Detail
	attendanceSessionGroup.Get("/", attendanceSessionController.ListClassAttendanceSessions)   // GET /admin/class-attendance-sessions
	attendanceSessionGroup.Get("/:id", attendanceSessionController.GetClassAttendanceSession) // GET /admin/class-attendance-sessions/:id
	
}