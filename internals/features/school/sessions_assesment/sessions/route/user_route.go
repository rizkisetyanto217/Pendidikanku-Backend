package route

import (
	uaCtrl "masjidku_backend/internals/features/school/sessions_assesment/sessions/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Contoh penggunaan middleware auth jika ada:
// import mw "masjidku_backend/internals/middlewares"

func AttendanceSessionsUserRoutes(r fiber.Router, db *gorm.DB) {
	// Attendance Sessions
	attendanceSessionController := uaCtrl.NewClassAttendanceSessionController(db)
	attendanceSessionGroup := r.Group("/class-attendance-sessions" /* , mw.AuthRequired() */)
	attendanceSessionGroup.Get("/", attendanceSessionController.ListClassAttendanceSessions)   // GET /.../class-attendance-sessions
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
	urlGroup := r.Group("/class-attendance-session-urls")
	urlGroup.Get("/filter", urlCtl.Filter) // list/filter
	urlGroup.Get("/:id", urlCtl.GetByID)   // detail by id

	// =====================
	// User Attendance URLs (read-only)  <-- DITAMBAHKAN
	// =====================
	uauCtl := uaCtrl.NewUserAttendanceUrlController(db)
	uauGroup := r.Group("/user-attendance-urls")
	uauGroup.Get("/", uauCtl.ListByAttendance) // ?attendance_id=...&limit=&offset=
	uauGroup.Get("/:id", uauCtl.GetByID)       // detail by id
}
