// file: internals/features/lembaga/masjid_yayasans/teachers_students/route/users_teacher_routes.go
package route

import (
	teacherController "masjidku_backend/internals/features/lembaga/masjid_yayasans/teachers_students/controller"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Mounted di group /api/u
func AllTeacherUserRoute(userRoute fiber.Router, db *gorm.DB) {
	v := validator.New()

	// Controllers
	tch := teacherController.NewMasjidTeacherController(db) // masjid_teachers
	std := teacherController.New(db, v)                     // masjid_students

	// ====== JOIN TEACHER (GLOBAL, tanpa masjid_id di path) ======
	// Body: { "code": "XXXXXX" } -> controller resolve masjid dari kode
	userRoute.Post("/join-teacher", tch.JoinAsTeacherWithCode)

	// ====== LIST (tetap scoped by masjid) ======
	// via :masjid_id
	mID := userRoute.Group("/:masjid_id")
	mID.Get("/masjid-teachers/list", tch.List)
	mID.Get("/masjid-students/list", std.List)

	// via :masjid_slug
	mSlug := userRoute.Group("/m/:masjid_slug")
	mSlug.Get("/masjid-teachers/list", tch.List)
	mSlug.Get("/masjid-students/list", std.List)
}
