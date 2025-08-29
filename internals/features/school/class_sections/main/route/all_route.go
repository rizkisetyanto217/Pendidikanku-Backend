// internals/features/lembaga/classes/sections/main/route/admin_route.go
package route

import (
	sectionctrl "masjidku_backend/internals/features/school/class_sections/main/controller"
	masjidkuMiddleware "masjidku_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func ClassSectionAllRoutes(r fiber.Router, db *gorm.DB) {
	h := sectionctrl.NewClassSectionController(db)

	// /admin/class-sections (semua pakai IsMasjidAdmin)
	sections := r.Group("/class-sections", masjidkuMiddleware.IsMasjidAdmin())

	sections.Get("/slug/:slug", h.GetClassSectionBySlug)
}
