// file: internals/features/school/sessions_assesment/sessions/route/teacher_routes.go
package route

import (
	uaCtrl "madinahsalam_backend/internals/features/school/classes/class_attendance_sessions/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AttendanceSessionsTeacherRoutes(r fiber.Router, db *gorm.DB) {
	// ✅ Base group tanpa :school_id
	base := r.Group("") // school context lewat helper di controller

	// Controller untuk sessions
	sessionController := uaCtrl.NewClassAttendanceSessionController(db)

	// =====================
	// Attendance Sessions
	// =====================
	sGroup := base.Group("/attendance-sessions")
	sGroup.Post("/", sessionController.CreateClassAttendanceSession)
	sGroup.Put("/:id", sessionController.PatchClassAttendanceSessionUrl)
	sGroup.Delete("/:id", sessionController.DeleteClassAttendanceSessionUrl)


	// =====================
	// User Attendance Types (CRUD)
	// =====================
	uattCtl := uaCtrl.NewClassAttendanceSessionParticipantTypeController(db)
	uatt := base.Group("/attendance-participant-types")
	uatt.Post("/", uattCtl.Create)
	uatt.Get("/", uattCtl.List)
	uatt.Patch("/:id", uattCtl.Patch)
	uatt.Delete("/:id", uattCtl.Delete)
	uatt.Post("/:id/restore", uattCtl.Restore)
}
