// file: internals/features/school/sessions_assesment/sessions/route/user_routes.go
package route

import (
	uaCtrl "schoolku_backend/internals/features/school/classes/class_attendance_sessions/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Contoh penggunaan middleware auth jika ada:
// import mw "schoolku_backend/internals/middlewares"
func AttendanceSessionsAdminRoutes(r fiber.Router, db *gorm.DB) {
	// ✅ Group dengan school context
	schoolGroup := r.Group("/:school_id")

	// =====================
	// User Attendance Types (CRUD)
	// =====================
	uattCtl := uaCtrl.NewClassAttendanceSessionParticipantTypeController(db)
	uatt := schoolGroup.Group("/attendance-participant-types")
	uatt.Post("/", uattCtl.Create)
	uatt.Get("/", uattCtl.List)
	uatt.Patch("/:id", uattCtl.Patch)
	uatt.Delete("/:id", uattCtl.Delete)
	uatt.Post("/:id/restore", uattCtl.Restore)
}
