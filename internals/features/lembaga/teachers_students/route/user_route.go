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
	ctl := teacherController.NewUsersTeacherController(db, v)
	std := teacherController.New(db, v)

	g := userRoute.Group("/user-teachers")
	g.Get("/by-user/:user_id", ctl.GetByUserID)
	g.Get("/", ctl.List)
	g.Get("/:id", ctl.GetByID)

	// ==== STUDENT READ-ONLY ====
	gs := userRoute.Group("/user-students")
	gs.Get("/", std.List)
	gs.Get("/:id", std.GetByID)
}
