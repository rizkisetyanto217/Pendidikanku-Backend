// file: internals/features/school/sessions_assesment/sessions/route/teacher_routes.go
package route

import (
	uaCtrl "masjidku_backend/internals/features/school/sessions/sessions/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)



func AttendanceSessionsTeacherRoutes(r fiber.Router, db *gorm.DB) {
	// Controller untuk sessions
	sessionController := uaCtrl.NewClassAttendanceSessionController(db)

	// =====================
	// Attendance Sessions
	// =====================
	sGroup := r.Group("/sessions")
	sGroup.Post("/", sessionController.CreateClassAttendanceSession)
	sGroup.Put("/:id", sessionController.UpdateClassAttendanceSession)
	sGroup.Delete("/:id", sessionController.DeleteClassAttendanceSession)
	sGroup.Get("/teacher/me", sessionController.ListMyTeachingSessions)
	sGroup.Get("/section/:section_id", sessionController.ListBySection)



	// =====================
	// Occurrences (Schedule & Attendance)
	// =====================
	occ := uaCtrl.NewOccurrenceController(db)
	// rencana (berulang mingguan → di-expand by date range)
	r.Get("/class-schedules/occurrences", occ.ListScheduleOccurrences)
	// sesi kehadiran aktual (langsung dari CAS)
	r.Get("/class-attendance-sessions/occurrences", occ.ListAttendanceOccurrences)


	uattCtl := uaCtrl.NewUserAttendanceTypeController(db)
	uatt := r.Group("/user-attendance-types")
	uatt.Post("/", uattCtl.Create)
	uatt.Get("/", uattCtl.List)
	uatt.Get("/:id", uattCtl.GetByID)
	uatt.Patch("/:id", uattCtl.Patch)
	uatt.Delete("/:id", uattCtl.Delete)
	uatt.Post("/:id/restore", uattCtl.Restore)
}
