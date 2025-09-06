package route

import (
	teacherController "masjidku_backend/internals/features/lembaga/teachers_students/controller"


	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func UsersTeacherRoute(app fiber.Router, db *gorm.DB) {
    v := validator.New()
    teacherCtrl := teacherController.NewUsersTeacherController(db, v)
    studentCtrl := teacherController.New(db, v) // reuse controller student (punya CRUD lengkap)

    // ===== TEACHER =====
    g := app.Group("/user-teachers")
    g.Post("/", teacherCtrl.Create)
    g.Get("/:id", teacherCtrl.GetByID)
    g.Get("/by-user/:user_id", teacherCtrl.GetByUserID)
    g.Get("/", teacherCtrl.List)
    g.Put("/:id", teacherCtrl.Update)
    g.Delete("/:id", teacherCtrl.Delete)

    // ===== STUDENT (BARU) =====
    s := app.Group("/user-students")
    s.Post("/", studentCtrl.Create)
    s.Get("/", studentCtrl.List)
    s.Get("/:id", studentCtrl.GetByID)
    s.Put("/:id", studentCtrl.Update)
    s.Patch("/:id", studentCtrl.Patch)
    s.Delete("/:id", studentCtrl.Delete)
    s.Post("/:id/restore", studentCtrl.Restore)
}
