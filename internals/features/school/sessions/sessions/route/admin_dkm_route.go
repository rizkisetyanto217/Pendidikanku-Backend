// file: internals/features/school/sessions_assesment/sessions/route/user_routes.go
package route

import (
	uaCtrl "masjidku_backend/internals/features/school/sessions/sessions/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Contoh penggunaan middleware auth jika ada:
// import mw "masjidku_backend/internals/middlewares"
func AttendanceSessionsAdminRoutes(r fiber.Router, db *gorm.DB) {
	// ✅ Group dengan masjid context
	masjidGroup := r.Group("/:masjid_id")

	// Attendance Sessions
	// attendanceSessionController := uaCtrl.NewClassAttendanceSessionController(db)
	// attendanceSessionGroup := masjidGroup.Group("/sessions" /* , mw.AuthRequired() */)
	// attendanceSessionGroup.Get("/list", attendanceSessionController.ListClassAttendanceSessions)  // GET /:masjid_id/.../class-attendance-sessions

	// User Attendance (read-only untuk user)
	// ua := uaCtrl.NewUserAttendanceController(db)
	// uaGroup := masjidGroup.Group("/user-attendance")
	// uaGroup.Get("/", ua.List)

	// =====================
	// Class Attendance Session URLs (read-only untuk user)
	// =====================
	// urlCtl := uaCtrl.NewClassAttendanceSessionURLController(db)
	// urlGroup := masjidGroup.Group("/session-urls")
	// urlGroup.Get("/filter", urlCtl.Filter) // list/filter

	// =====================
	// Occurrences (Schedule & Attendance)
	// =====================
	// occ := uaCtrl.NewOccurrenceController(db)
	// // rencana (berulang mingguan → di-expand by date range)
	// masjidGroup.Get("/class-schedules/occurrences", occ.ListScheduleOccurrences)
	// // sesi kehadiran aktual (langsung dari CAS)
	// masjidGroup.Get("/class-attendance-sessions/occurrences", occ.ListAttendanceOccurrences)

	// =====================
	// User Attendance Types (CRUD)
	// =====================
	uattCtl := uaCtrl.NewUserAttendanceTypeController(db)
	uatt := masjidGroup.Group("/user-attendance-types")
	uatt.Post("/", uattCtl.Create)
	uatt.Get("/", uattCtl.List)
	uatt.Get("/:id", uattCtl.GetByID)
	uatt.Patch("/:id", uattCtl.Patch)
	uatt.Delete("/:id", uattCtl.Delete)
	uatt.Post("/:id/restore", uattCtl.Restore)
}
