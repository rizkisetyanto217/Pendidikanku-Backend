// internals/route/classes_admin_routes.go
package route

import (
	classctrl "masjidku_backend/internals/features/school/classes/classes/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func ClassAllRoutes(admin fiber.Router, db *gorm.DB) {
	h := classctrl.NewClassController(db)

	// /admin/classes (semua pakai IsMasjidAdmin)
	classes := admin.Group("/classes")
	classes.Get("/slug/:slug", h.GetClassBySlug)
}