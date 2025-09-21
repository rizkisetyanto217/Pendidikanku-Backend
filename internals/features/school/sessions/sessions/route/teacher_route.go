// file: internals/features/school/sessions_assesment/sessions/route/teacher_routes.go
package route

import (
	uaCtrl "masjidku_backend/internals/features/school/sessions/sessions/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AttendanceSessionsTeacherRoutes(r fiber.Router, db *gorm.DB) {
	// ✅ Group dengan masjid context
	masjidGroup := r.Group("/:masjid_id")

	// Controller untuk sessions
	sessionController := uaCtrl.NewClassAttendanceSessionController(db)

	// =====================
	// Attendance Sessions
	// =====================
	sGroup := masjidGroup.Group("/sessions")
	sGroup.Post("/", sessionController.CreateClassAttendanceSession)
	sGroup.Put("/:id", sessionController.PatchClassAttendanceSessionUrl)
	sGroup.Delete("/:id", sessionController.DeleteClassAttendanceSessionUrl)
	sGroup.Get("/teacher/me", sessionController.ListMyTeachingSessions)

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
