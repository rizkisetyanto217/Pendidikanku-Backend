// file: internals/features/users/user_profiles/route/teacher_route.go
package routes

import (
	teacherController "masjidku_backend/internals/features/lembaga/masjid_teachers/teachers/controller"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func UsersTeacherRoute(app fiber.Router, db *gorm.DB) {
    v := validator.New()
    teacherCtrl := teacherController.NewUsersTeacherController(db, v)

    g := app.Group("/user-teachers")

    g.Post("/", teacherCtrl.Create)
    g.Get("/:id", teacherCtrl.GetByID)
    g.Get("/by-user/:user_id", teacherCtrl.GetByUserID)
    g.Get("/", teacherCtrl.List)
    g.Put("/:id", teacherCtrl.Update)
    g.Delete("/:id", teacherCtrl.Delete)
}
