// internals/route/details/lembaga_details_routes.go
package details

import (
	// ====== Lembaga features ======
	ClassSectionsRoutes "masjidku_backend/internals/features/lembaga/class_sections/main/route"
	ClassesRoutes "masjidku_backend/internals/features/lembaga/classes/main/route"

	// Tambahkan import route lain di sini saat modul siap:
	// SectionRoutes "masjidku_backend/internals/features/lembaga/sections/main/route"
	// StudentRoutes "masjidku_backend/internals/features/lembaga/students/main/route"
	// TeacherRoutes "masjidku_backend/internals/features/lembaga/teachers/main/route"
	// ScheduleRoutes "masjidku_backend/internals/features/lembaga/schedules/main/route"
	// FinanceRoutes  "masjidku_backend/internals/features/lembaga/finance/main/route"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

/* ===================== PUBLIC ===================== */
// Endpoint publik (boleh diakses tanpa login, atau pakai SecondAuth untuk optional user)
func LembagaPublicRoutes(r fiber.Router, db *gorm.DB) {
	ClassesRoutes.ClassAllRoutes(r, db)
	ClassSectionsRoutes.ClassSectionAllRoutes(r, db)
	// Classes (public read)
	// ClassRoutes.ClassPublicRoutes(r, db)

	// Tambahkan modul lain (public) di sini:
	// SectionRoutes.SectionPublicRoutes(r, db)
	// ScheduleRoutes.SchedulePublicRoutes(r, db)
}

/* ===================== USER (PRIVATE) ===================== */
// Endpoint yang butuh login user biasa (token user)
func LembagaUserRoutes(r fiber.Router, db *gorm.DB) {
	ClassesRoutes.UserClassesStudentRoutes(r, db)
	// Classes (user actions: enroll, progress, dsb)
	// ClassRoutes.ClassUserRoutes(r, db)

	// Tambahkan modul lain (user) di sini:
	// StudentRoutes.StudentUserRoutes(r, db)
	// ScheduleRoutes.ScheduleUserRoutes(r, db)
}

/* ===================== ADMIN ===================== */
// Endpoint khusus admin lembaga/masjid (token + guard admin)
func LembagaAdminRoutes(r fiber.Router, db *gorm.DB) {
	// Classes (CRUD admin)
	ClassesRoutes.ClassAdminRoutes(r, db)
	ClassSectionsRoutes.ClassSectionAdminRoutes(r, db)

	// Tambahkan modul lain (admin) di sini:
	// SectionRoutes.SectionAdminRoutes(r, db)
	// TeacherRoutes.TeacherAdminRoutes(r, db)
	// FinanceRoutes.FinanceAdminRoutes(r, db)
}
