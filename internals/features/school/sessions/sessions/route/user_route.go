// file: internals/features/school/sessions_assesment/sessions/route/user_routes.go
package route

import (
	uaCtrl "masjidku_backend/internals/features/school/sessions/sessions/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Contoh penggunaan middleware auth jika ada:
// import mw "masjidku_backend/internals/middlewares"

func AttendanceSessionsUserRoutes(r fiber.Router, db *gorm.DB) {
	// Attendance Sessions
	attendanceSessionController := uaCtrl.NewClassAttendanceSessionController(db)
	attendanceSessionGroup := r.Group("/sessions" /* , mw.AuthRequired() */)
	attendanceSessionGroup.Get("/list", attendanceSessionController.ListClassAttendanceSessions)  // GET /.../class-attendance-sessions


	// User Attendance (read-only untuk user)
	ua := uaCtrl.NewUserAttendanceController(db)
	uaGroup := r.Group("/user-sessions")
	uaGroup.Get("/list", ua.List)
	uaGroup.Post("/", ua.Create)      // create
	uaGroup.Patch("/:id", ua.Update)  // partial update
	uaGroup.Delete("/:id", ua.Delete) // soft delete

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
	uatt.Get("/", uattCtl.List)
	uatt.Get("/:id", uattCtl.GetByID)
}
