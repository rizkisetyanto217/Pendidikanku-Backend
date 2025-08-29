// internals/features/lembaga/classes/user_classes/main/route/user_routes.go
package route

import (
	ctrl "masjidku_backend/internals/features/school/classes/main/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func ClassUserRoutes(r fiber.Router, db *gorm.DB) {
	h := ctrl.NewUserMyClassController(db)

	g := r.Group("/user-classes")
	g.Get("/", h.ListMyUserClasses)      // GET list enrolment milik user
	g.Get("/:id", h.GetMyUserClassByID)  // GET detail enrolment milik user
	g.Post("/", h.SelfEnroll)            // PMB: daftar kelas (status=inactive)

	// ===== CPO (Class Pricing Options) - USER (read-only) =====
	cpo := ctrl.NewCPOController(db)

	// by class_id
	r.Get("/classes/:class_id/pricing-options", cpo.UserListCPO)     // ?type=&limit=&offset=
	r.Get("/classes/:class_id/pricing-options/latest", cpo.UserLatestCPO)

	// by pricing option id
	r.Get("/pricing-options/:id", cpo.UserGetCPOByID)
}
