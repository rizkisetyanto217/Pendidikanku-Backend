// file: internals/features/school/sessions_assesment/sessions/route/user_routes.go
package route

import (
	uaCtrl "schoolku_backend/internals/features/school/classes/class_attendance_sessions/controller"

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
}
