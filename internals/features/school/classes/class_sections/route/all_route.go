// internals/features/lembaga/classes/sections/main/route/admin_route.go
package route

import (
	sectionctrl "masjidku_backend/internals/features/school/classes/class_sections/controller"
	masjidkuMiddleware "masjidku_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AllClassSectionRoutes(r fiber.Router, db *gorm.DB) {
	h := sectionctrl.NewClassSectionController(db)

	// ===== Base by masjid_id =====
	baseByID := r.Group("/:masjid_id")
	sectionsByID := baseByID.Group("/class-sections")
	sectionsByID.Get("/list", h.ListClassSections)

	// ===== Base by masjid_slug (beri prefix 'slug' agar tidak bentrok) =====
	baseBySlug := r.Group("/slug/:masjid_slug", masjidkuMiddleware.UseMasjidScope())
	sectionsBySlug := baseBySlug.Group("/class-sections")
	sectionsBySlug.Get("/list", h.ListClassSections)
}
