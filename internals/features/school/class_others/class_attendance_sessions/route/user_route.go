package route

import (
	uaCtrl "madinahsalam_backend/internals/features/school/class_others/class_attendance_sessions/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AttendanceSessionsUserRoutes(r fiber.Router, db *gorm.DB) {
	// Attendance Sessions (read-only untuk user)
	attendanceSessionController := uaCtrl.NewClassAttendanceSessionController(db)
	asg := r.Group("/attendance-sessions")
	asg.Get("/list", attendanceSessionController.ListClassAttendanceSessions)

	// Attendance Participants (user CRUD)
	ua := uaCtrl.NewClassAttendanceSessionParticipantController(db)
	uag := r.Group("/attendance-participants")
	uag.Get("/list", ua.List)
	uag.Post("/", ua.CreateAttendanceParticipantsWithURLs)
	uag.Patch("/:id", ua.Patch)
	uag.Delete("/:id", ua.Delete)

	// Attendance Participant Types (read-only)
	uattCtl := uaCtrl.NewClassAttendanceSessionParticipantTypeController(db)
	utg := r.Group("/attendance-participant-types")
	utg.Get("/", uattCtl.List)

	// ============================
	// Attendance Session Types (read-only untuk user)
	// ============================
	stCtl := uaCtrl.NewClassAttendanceSessionTypeController(db)
	stg := r.Group("/attendance-session-types")

	// konsisten pakai /list seperti yang lain
	stg.Get("/list", stCtl.List)
}
