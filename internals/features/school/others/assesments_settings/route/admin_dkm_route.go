package route

import (
	settingCtrl "madinahsalam_backend/internals/features/school/others/assesments_settings/controller"

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
