package route

import (
	uaCtrl "madinahsalam_backend/internals/features/school/classes/class_attendance_sessions/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Contoh penggunaan middleware auth jika ada:
// import mw "madinahsalam_backend/internals/middlewares"
func AttendanceSessionsAdminRoutes(r fiber.Router, db *gorm.DB) {
	// ✅ Base group tanpa :school_id
	//    school_id diambil dari context/token di controller (UseSchoolScope middleware, dll)
	base := r.Group("")

	// =====================
	// Attendance Participant Types (CRUD)
	// =====================
	uattCtl := uaCtrl.NewClassAttendanceSessionParticipantTypeController(db)
	uatt := base.Group("/attendance-participant-types")

	uatt.Post("/", uattCtl.Create)
	uatt.Get("/", uattCtl.List)
	uatt.Patch("/:id", uattCtl.Patch)
	uatt.Delete("/:id", uattCtl.Delete)
	uatt.Post("/:id/restore", uattCtl.Restore)

	// =====================
	// Attendance Session Types (CRUD master per tenant)
	// =====================
	stCtl := uaCtrl.NewClassAttendanceSessionTypeController(db)
	st := base.Group("/attendance-session-types")

	// create + list
	st.Post("/", stCtl.Create)
	st.Get("/", stCtl.List)

	// detail + update + delete
	st.Get("/:id", stCtl.Detail)
	st.Put("/:id", stCtl.Update)
	st.Delete("/:id", stCtl.Delete)
}
