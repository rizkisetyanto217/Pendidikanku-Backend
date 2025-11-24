// internals/route/details/lembaga_details_routes.go
package details

import (
	// ====== Lembaga features ======

	// CertificateRoutes "madinahsalam_backend/internals/features/school/academics/certificates/route"

	ScheduleRoutes "madinahsalam_backend/internals/features/school/classes/class_schedules/route"

	LembagaRoutes "madinahsalam_backend/internals/features/lembaga/school_yayasans/schools/route"

	LembagaSchoolTeacher "madinahsalam_backend/internals/features/lembaga/school_yayasans/teachers_students/route"

	// Tambahkan import route lain di sini saat modul siap:
	// SectionRoutes "madinahsalam_backend/internals/features/lembaga/sections/main/route"
	// StudentRoutes "madinahsalam_backend/internals/features/lembaga/students/main/route"
	// TeacherRoutes "madinahsalam_backend/internals/features/lembaga/teachers/main/route"
	// ScheduleRoutes "madinahsalam_backend/internals/features/lembaga/schedules/main/route"
	// FinanceRoutes  "madinahsalam_backend/internals/features/lembaga/finance/main/route"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

/* ===================== PUBLIC ===================== */
// Endpoint publik (boleh diakses tanpa login, atau pakai SecondAuth untuk optional user)
func LembagaPublicRoutes(r fiber.Router, db *gorm.DB) {
	LembagaRoutes.AllLembagaRoutes(r, db)
	LembagaSchoolTeacher.AllLembagaTeacherStudentRoutes(r, db)
}

/* ===================== USER (PRIVATE) ===================== */
// Endpoint yang butuh login user biasa (token user)
func LembagaUserRoutes(r fiber.Router, db *gorm.DB) {

	ScheduleRoutes.ScheduleUserRoutes(r, db)
	LembagaSchoolTeacher.LembagaTeacherStudentUserRoutes(r, db)

}

/* ===================== ADMIN ===================== */
// Endpoint khusus admin lembaga/school (token + guard admin)
func LembagaAdminRoutes(r fiber.Router, db *gorm.DB) {
	LembagaRoutes.SchoolAdminRoutes(r, db)
	LembagaSchoolTeacher.LembagaTeacherStudentAdminRoutes(r, db)
}

/* ===================== SUPER ADMIN ===================== */
// Endpoint khusus super admin (token + guard super admin)
func LembagaOwnerRoutes(r fiber.Router, db *gorm.DB) {
	LembagaRoutes.SchoolOwnerRoutes(r, db)
}
