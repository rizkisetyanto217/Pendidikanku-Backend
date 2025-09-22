package route

import (
	teacherController "masjidku_backend/internals/features/lembaga/teachers_students/controller"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func UsersTeacherUserRoute(userRoute fiber.Router, db *gorm.DB) {
	v := validator.New()

	// ❌ WAS: NewUserTeacherController (mengarah ke tabel user_teachers)
	// ctl := teacherController.NewUserTeacherController(db, v)

	// ✅ USE: MasjidTeacherController (mengarah ke tabel masjid_teachers)
	tch := teacherController.NewMasjidTeacherController(db)

	// Student controller (sudah oke)
	std := teacherController.New(db, v)

	// ===== pakai :masjid_id =====
	mID := userRoute.Group("/:masjid_id")
	mID.Get("/masjid-teachers/list", tch.List)
	mID.Get("/masjid-students/list", std.List)

	// ===== (opsional) pakai :masjid_slug =====
	mSlug := userRoute.Group("/m/:masjid_slug")
	mSlug.Get("/masjid-teachers/list", tch.List)
	mSlug.Get("/masjid-students/list", std.List)
}
