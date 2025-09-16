package route

import (
	teacherController "masjidku_backend/internals/features/lembaga/teachers_students/controller"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Routes untuk USER (read-only / publik / logged-in biasa).
// Contoh mount:
//   user := api.Group("/api/u")
//   routes.UsersTeacherUserRoute(user, db)
func UsersTeacherUserRoute(userRoute fiber.Router, db *gorm.DB) {
	v := validator.New()
	ctl := teacherController.NewUserTeacherController(db, v)
	std := teacherController.New(db, v)

	g := userRoute.Group("/user-teachers")
	g.Get("/list", ctl.List)

	// ==== STUDENT READ-ONLY ====
	gs := userRoute.Group("/masjid-students")
	gs.Get("/list", std.List)
}
