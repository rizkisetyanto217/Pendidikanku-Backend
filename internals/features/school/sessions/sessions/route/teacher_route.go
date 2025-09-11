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
	// User Attendance (CRUD)
	// =====================
	ua := uaCtrl.NewUserAttendanceController(db)
	uaGroup := r.Group("/user-sessions")
	uaGroup.Get("/list", ua.List)         // list tenant-safe
	uaGroup.Post("/", ua.Create)      // create
	uaGroup.Get("/:id", ua.GetByID)   // detail
	uaGroup.Patch("/:id", ua.Update)  // partial update
	uaGroup.Delete("/:id", ua.Delete) // soft delete

	// =====================
	// Class Attendance Session URLs (CRUD)
	// =====================
	urlCtl := uaCtrl.NewClassAttendanceSessionURLController(db)
	urlGroup := r.Group("/session-urls")
	urlGroup.Post("/", urlCtl.Create)      // create (JSON atau multipart file)
	urlGroup.Patch("/:id", urlCtl.Update)  // update (JSON atau multipart file)
	urlGroup.Get("/:id", urlCtl.GetByID)   // detail
	urlGroup.Get("/filter", urlCtl.Filter) // list/filter
	urlGroup.Delete("/:id", urlCtl.Delete) // soft delete (+ move to spam)

	// =====================
	// User Attendance URLs (CRUD)
	// =====================
	uauCtl := uaCtrl.NewUserAttendanceUrlController(db)
	uauGroup := r.Group("/user-session-urls")
	uauGroup.Post("/", uauCtl.CreateJSON)             // create via JSON (href langsung)
	uauGroup.Post("/multipart", uauCtl.CreateMultipart) // create via multipart (upload -> OSS -> href)
	uauGroup.Patch("/:id", uauCtl.Update)             // update (JSON atau multipart)
	uauGroup.Get("/:id", uauCtl.GetByID)              // detail by id
	uauGroup.Get("/", uauCtl.ListByAttendance)        // ?attendance_id=...&limit=&offset=
	uauGroup.Delete("/:id", uauCtl.SoftDelete)        // soft delete

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
