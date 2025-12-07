// file: internals/features/school/sessions_assesment/sessions/route/teacher_routes.go
package route

import (
	attendanceParticipantController "madinahsalam_backend/internals/features/school/class_others/class_attendance_sessions/controller/participants"
	attendanceController "madinahsalam_backend/internals/features/school/class_others/class_attendance_sessions/controller/sessions"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AttendanceSessionsTeacherRoutes(r fiber.Router, db *gorm.DB) {
	// ✅ Base group tanpa :school_id
	base := r.Group("") // school context lewat helper di controller

	// Controller untuk sessions
	sessionController := attendanceController.NewClassAttendanceSessionController(db)

	// =====================
	// Attendance Sessions
	// =====================
	sGroup := base.Group("/attendance-sessions")

	// GET pakai controller baru ListClassAttendanceSessions
	sGroup.Get("/", sessionController.ListClassAttendanceSessions)

	// POST pakai controller baru CreateClassAttendanceSession
	sGroup.Post("/", sessionController.CreateClassAttendanceSession)

	// (PUT/DELETE URL sudah tidak dipakai di versi baru, karena URL di-handle via
	//  DTO UpdateClassAttendanceSessionRequest: urls_add / urls_patch / urls_delete)
	// Kalau nanti kamu bikin endpoint PATCH session, tinggal tambahin:
	// sGroup.Patch("/:id", sessionController.PatchClassAttendanceSession)
	// dan implement method-nya di controller.
	// Kalau mau delete session:
	// sGroup.Delete("/:id", sessionController.DeleteClassAttendanceSession)

	// =====================
	// User Attendance Types (CRUD)
	// =====================
	uattCtl := attendanceParticipantController.NewClassAttendanceSessionParticipantTypeController(db)
	uatt := base.Group("/attendance-participant-types")
	uatt.Post("/", uattCtl.Create)
	uatt.Get("/", uattCtl.List)
	uatt.Patch("/:id", uattCtl.Patch)
	uatt.Delete("/:id", uattCtl.Delete)
	uatt.Post("/:id/restore", uattCtl.Restore)
}
