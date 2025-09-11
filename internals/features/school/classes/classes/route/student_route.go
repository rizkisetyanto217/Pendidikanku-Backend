// internals/features/lembaga/classes/user_classes/main/route/user_routes.go
package route

import (
	ctrl "masjidku_backend/internals/features/school/classes/classes/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func ClassUserRoutes(r fiber.Router, db *gorm.DB) {
	// ===== Classes (READ-ONLY untuk user) =====
	cls := ctrl.NewClassController(db)

	classes := r.Group("/classes")
	classes.Get("/list", cls.ListClasses)    // list kelas (read-only)
	classes.Get("/search", cls.SearchWithSubjects)
	classes.Get("/slug/:slug", cls.GetClassBySlug)

	// ===== Class Parents (READ-ONLY untuk user) =====
	cp := ctrl.NewClassParentController(db, nil)
	classParents := r.Group("/class-parents")
	classParents.Get("/list", cp.List)  // list parent (read-only)
	classParents.Get("/:id", cp.GetByID)

	// ===== "My User Classes" (enrolment milik user) =====
	my := ctrl.NewUserMyClassController(db)
	uc := r.Group("/user-classes")
	uc.Get("/", my.ListMyUserClasses)     // GET list enrolment milik user
	uc.Get("/:id", my.GetMyUserClassByID) // GET detail enrolment milik user
	uc.Post("/", my.SelfEnroll)           // PMB: daftar kelas (status=inactive, by pricing option id)
}
