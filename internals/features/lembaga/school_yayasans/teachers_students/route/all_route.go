// file: internals/features/lembaga/school_yayasans/teachers_students/route/users_teacher_routes.go
package route

import (
	teacherController "schoolku_backend/internals/features/lembaga/school_yayasans/teachers_students/controller"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Mounted di group /api/u
func AllTeacherUserRoute(userRoute fiber.Router, db *gorm.DB) {
	v := validator.New()

	// Controllers
	tch := teacherController.NewSchoolTeacherController(db) // school_teachers
	std := teacherController.New(db, v)                     // school_students

	// ====== JOIN TEACHER (GLOBAL, tanpa school_id di path) ======
	// Body: { "code": "XXXXXX" } -> controller resolve school dari kode
	userRoute.Post("/join-teacher", tch.JoinAsTeacherWithCode)

	// ====== LIST (tetap scoped by school) ======
	// via :school_id
	mID := userRoute.Group("/:school_id")
	mID.Get("/school-teachers/list", tch.List)
	mID.Get("/school-students/list", std.List)

	// via :school_slug
	mSlug := userRoute.Group("/m/:school_slug")
	mSlug.Get("/school-teachers/list", tch.List)
	mSlug.Get("/school-students/list", std.List)
}
