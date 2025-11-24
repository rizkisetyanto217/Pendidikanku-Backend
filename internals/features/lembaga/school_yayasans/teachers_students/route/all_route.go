// file: internals/features/lembaga/school_yayasans/teachers_students/route/users_teacher_routes.go
package route

import (
	teacherController "madinahsalam_backend/internals/features/lembaga/school_yayasans/teachers_students/controller"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Mounted di group /api/u
func AllLembagaTeacherStudentRoutes(userRoute fiber.Router, db *gorm.DB) {
	v := validator.New()

	// Controllers
	tch := teacherController.NewSchoolTeacherController(db) // school_teachers
	std := teacherController.New(db, v)                     // school_students

	// ====== JOIN TEACHER (GLOBAL, tanpa school_id di path) ======
	// Body: { "code": "XXXXXX" } -> controller resolve school dari kode
	userRoute.Post("/join-teacher", tch.JoinAsTeacherWithCode)

	// ====== LIST (school dari TOKEN, bukan dari path) ======
	// Controller List diharapkan pakai:
	//   - helperAuth.ResolveSchoolIDFromContext(c)
	//   - helperAuth.EnsureMemberSchool(c, schoolID) atau EnsureTeacherSchool, dsb
	userRoute.Get("/school-teachers/list", tch.List)
	userRoute.Get("/school-students/list", std.List)
}
