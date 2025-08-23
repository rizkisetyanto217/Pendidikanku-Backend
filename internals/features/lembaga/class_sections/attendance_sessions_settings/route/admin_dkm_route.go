package route

import (
	settingCtrl "masjidku_backend/internals/features/lembaga/class_sections/attendance_sessions_settings/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func ClassAttendanceSettingsAdminRoutes(r fiber.Router, db *gorm.DB) {
	// Controller
	ctrl := settingCtrl.NewClassAttendanceSettingController(db)

	// ==========================
	// Attendance Settings (Admin)
	// ==========================
	sGroup := r.Group("/class_attendance_settings")
	// Ambil setting global per sekolah
	sGroup.Post("/", ctrl.CreateBySchool)
	sGroup.Get("/", ctrl.GetBySchool)
	// Upsert setting global per sekolah
	sGroup.Put("/", ctrl.UpdateBySchool)
}
