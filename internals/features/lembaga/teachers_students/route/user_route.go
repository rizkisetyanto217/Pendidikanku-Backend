package route

import (
	teacherController "masjidku_backend/internals/features/lembaga/teachers_students/controller"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Mounted di group /api/u
func UsersTeacherUserRoute(userRoute fiber.Router, db *gorm.DB) {
	v := validator.New()

	// Controllers

	tch := teacherController.NewMasjidTeacherController(db) // list masjid_teachers
	std := teacherController.New(db, v)                     // student controller kamu yang sudah ada

	// ====== MASJID_TEACHERS (LIST) ======
	// via :masjid_id
	mID := userRoute.Group("/:masjid_id")
	mID.Get("/masjid-teachers/list", tch.List)
	mID.Get("/masjid-students/list", std.List)

	// via :masjid_slug
	mSlug := userRoute.Group("/m/:masjid_slug")
	mSlug.Get("/masjid-teachers/list", tch.List)
	mSlug.Get("/masjid-students/list", std.List)
}
