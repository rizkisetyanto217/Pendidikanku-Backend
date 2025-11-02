// file: internals/features/school/sessions_assesment/sessions/route/teacher_routes.go
package route

import (
	uaCtrl "schoolku_backend/internals/features/school/classes/class_attendance_sessions/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AttendanceSessionsTeacherRoutes(r fiber.Router, db *gorm.DB) {
	// ✅ Group dengan school context
	schoolGroup := r.Group("/:school_id")

	// Controller untuk sessions
	sessionController := uaCtrl.NewClassAttendanceSessionController(db)

	// =====================
	// Attendance Sessions
	// =====================
	sGroup := schoolGroup.Group("/sessions")
	sGroup.Post("/", sessionController.CreateClassAttendanceSession)
	sGroup.Put("/:id", sessionController.PatchClassAttendanceSessionUrl)
	sGroup.Delete("/:id", sessionController.DeleteClassAttendanceSessionUrl)
	sGroup.Get("/teacher/me", sessionController.ListMyTeachingSessions)

	// =====================
	// User Attendance Types (CRUD)
	// =====================
	uattCtl := uaCtrl.NewStudentAttendanceTypeController(db)
	uatt := schoolGroup.Group("/student-attendance-types")
	uatt.Post("/", uattCtl.Create)
	uatt.Get("/", uattCtl.List)
	uatt.Patch("/:id", uattCtl.Patch)
	uatt.Delete("/:id", uattCtl.Delete)
	uatt.Post("/:id/restore", uattCtl.Restore)
}
