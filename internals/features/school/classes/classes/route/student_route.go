// internals/features/lembaga/classes/user_classes/main/route/user_routes.go
package route

import (
	ctrl "schoolku_backend/internals/features/school/classes/classes/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func ClassUserRoutes(r fiber.Router, db *gorm.DB) {
	// ===== Classes (READ-ONLY untuk user) =====
	cls := ctrl.NewClassController(db)
	// Tenant-aware prefix
	classes := r.Group("/:school_id/classes")
	classes.Get("/list", cls.ListClasses) // list kelas (read-only)
	classes.Get("/slug/:slug", cls.GetClassBySlug)

}
