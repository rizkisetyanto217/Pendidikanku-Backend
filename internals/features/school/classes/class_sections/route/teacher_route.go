package route

import (
	classSectionStudentsController "madinahsalam_backend/internals/features/school/classes/class_sections/controller/student_class_sections"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

/*
Route khusus STAFF AKADEMIK (Guru / DKM / Admin / Bendahara)

- Pakai :school_id di path (tenant-safe, cocok dengan controller kamu).
- Controller sudah handle:
  - parseSchoolIDFromPath → baca :school_id
  - EnsureDKMOrTeacherSchool / EnsureStaffSchool untuk role check.

- Frontend guru tinggal pastikan :school_id = active_school_id dari token.
*/
func StudentClassSectionTeacherRoutes(r fiber.Router, db *gorm.DB) {
	ucsH := classSectionStudentsController.NewStudentClassSectionController(db)

	// Contoh mount di main:
	// apiT := app.Group("/api/t")
	// StudentClassSectionTeacherRoutes(apiT, db)
	//
	// Hasil endpoint:
	//   POST   /api/t/:school_id/student-class-sections
	//   GET    /api/t/:school_id/student-class-sections/:id
	//   PATCH  /api/t/:school_id/student-class-sections/:id
	//   DELETE /api/t/:school_id/student-class-sections/:id

	staff := r.Group("/:school_id/student-class-sections")

	// CREATE relasi murid ↔ section (staff only: DKM / Guru / Admin)
	staff.Post("/", ucsH.Create)

	// DETAIL satu relasi murid ↔ section (staff only)
	staff.Get("/:id", ucsH.GetDetail)

	// PATCH relasi murid ↔ section (staff only)
	staff.Patch("/:id", ucsH.Patch)

	// SOFT DELETE relasi murid ↔ section (staff only)
	staff.Delete("/:id", ucsH.Delete)
}
