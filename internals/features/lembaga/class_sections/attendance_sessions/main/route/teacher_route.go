// internals/features/lembaga/class_sections/attendance_sessions/main/router/class_attendance_teacher_routes.go
package route

import (
	entryCtrl "masjidku_backend/internals/features/lembaga/class_sections/attendance_sessions/main/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func ClassAttendanceSessionsTeacherRoutes(r fiber.Router, db *gorm.DB) {
	// Controller untuk sessions
	sessionController := entryCtrl.NewClassAttendanceSessionController(db)

	// Controller untuk entries
	entryController := entryCtrl.NewTeacherClassAttendanceSessionController(db)

	// =====================
	// Attendance Sessions
	// =====================
	sGroup := r.Group("/class-attendance-sessions")
	sGroup.Post("/", sessionController.CreateClassAttendanceSession)
	sGroup.Put("/:id", sessionController.UpdateClassAttendanceSession)
	sGroup.Delete("/:id", sessionController.DeleteClassAttendanceSession)

	// =====================
	// Attendance Entries
	// =====================
	eGroup := r.Group("/user_class_attendance_entries")
	eGroup.Post("/", entryController.CreateAttendanceSession)
	eGroup.Get("/", entryController.ListAttendanceSessions)
	eGroup.Patch("/:id", entryController.UpdateAttendanceSession)
}
