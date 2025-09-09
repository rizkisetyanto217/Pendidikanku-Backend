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
	attendanceSessionGroup.Get("/:id", attendanceSessionController.GetClassAttendanceSession) // GET /.../class-attendance-sessions/:id

	// User Attendance (read-only untuk user)
	ua := uaCtrl.NewUserAttendanceController(db)
	uaGroup := r.Group("/user-attendance")
	uaGroup.Get("/", ua.List)
	uaGroup.Get("/:id", ua.GetByID)

	// =====================
	// Class Attendance Session URLs (read-only untuk user)
	// =====================
	urlCtl := uaCtrl.NewClassAttendanceSessionURLController(db)
	urlGroup := r.Group("/session-urls")
	urlGroup.Get("/filter", urlCtl.Filter) // list/filter
	urlGroup.Get("/:id", urlCtl.GetByID)   // detail by id

	// =====================
	// User Attendance URLs (read-only)
	// =====================
	uauCtl := uaCtrl.NewUserAttendanceUrlController(db)
	uauGroup := r.Group("/user-attendance-urls")
	uauGroup.Get("/", uauCtl.ListByAttendance) // ?attendance_id=...&limit=&offset=
	uauGroup.Get("/:id", uauCtl.GetByID)       // detail by id

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
