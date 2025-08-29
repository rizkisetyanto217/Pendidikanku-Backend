// internals/features/lembaga/class_sections/attendance_sessions/main/router/class_attendance_teacher_routes.go
package route

import (
	entryCtrl "masjidku_backend/internals/features/school/class_attendance_result/attendance_sessions/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AttendanceSessionsTeacherRoutes(r fiber.Router, db *gorm.DB) {
	// Controller untuk sessions
	sessionController := entryCtrl.NewClassAttendanceSessionController(db)

	// Controller untuk entries
	entryController := entryCtrl.NewTeacherClassAttendanceSessionController(db)

	// =====================
	// Attendance Sessions
	// =====================
	sGroup := r.Group("/class-attendance-sessions")
	sGroup.Get("/by-masjid", sessionController.ListByMasjid)
	sGroup.Post("/", sessionController.CreateClassAttendanceSession)
	sGroup.Put("/:id", sessionController.UpdateClassAttendanceSession)
	sGroup.Delete("/:id", sessionController.DeleteClassAttendanceSession)
	sGroup.Get("/teacher/me", sessionController.ListMyTeachingSessions)  
	sGroup.Get("/section/:section_id", sessionController.ListBySection)

	// =====================
	// Attendance Entries
	// =====================
	eGroup := r.Group("/user_class_attendance")
	eGroup.Post("/", entryController.CreateAttendanceSession)
	eGroup.Get("/", entryController.ListAttendanceSessions)
	eGroup.Put("/:id", entryController.UpdateAttendanceSession)
}
