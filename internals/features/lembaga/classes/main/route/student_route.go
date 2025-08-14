// internals/features/lembaga/classes/user_classes/main/route/user_routes.go
package route

import (
	ctrl "masjidku_backend/internals/features/lembaga/classes/main/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func UserClassesStudentRoutes(r fiber.Router, db *gorm.DB) {
	h := ctrl.NewUserMyClassController(db)

	g := r.Group("/user-classes")
	g.Get("/", h.ListMyUserClasses)   // GET list enrolment milik user
	g.Get("/:id", h.GetMyUserClassByID) // GET detail enrolment milik user
	g.Post("/", h.SelfEnroll)         // PMB: daftar kelas (status=inactive)
}