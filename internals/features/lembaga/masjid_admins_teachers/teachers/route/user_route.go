// internals/features/users/user_profiles/routes/users_teacher_routes.go
package routes

import (
	teacherController "masjidku_backend/internals/features/lembaga/masjid_admins_teachers/teachers/controller"

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

	g := userRoute.Group("/user-teachers")

	// Route dengan segmen statis didaftarkan dulu (aman di Fiber, tapi lebih jelas).
	g.Get("/by-user/:user_id", ctl.GetByUserID)

	// List & detail
	g.Get("/", ctl.List)
	g.Get("/:id", ctl.GetByID)
}
