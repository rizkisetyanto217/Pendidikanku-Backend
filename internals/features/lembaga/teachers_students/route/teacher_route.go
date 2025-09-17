package route

import (
	teacherController "masjidku_backend/internals/features/lembaga/teachers_students/controller"


	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func UsersTeacherRoute(app fiber.Router, db *gorm.DB) {
    v := validator.New()
    teacherCtrl := teacherController.NewUserTeacherController(db, v)
    studentCtrl := teacherController.New(db, v) // reuse controller student (punya CRUD lengkap)

    // ===== TEACHER =====
    g := app.Group("/user-teachers")
    g.Post("/", teacherCtrl.Create)
    g.Put("/:id", teacherCtrl.Update)
    g.Delete("/:id", teacherCtrl.Delete)

    // ===== STUDENT (BARU) =====
    s := app.Group("/masjid-students")
    s.Post("/", studentCtrl.Create)
    s.Put("/:id", studentCtrl.Update)
    s.Patch("/:id", studentCtrl.Patch)
    s.Delete("/:id", studentCtrl.Delete)
    s.Post("/:id/restore", studentCtrl.Restore)
}
