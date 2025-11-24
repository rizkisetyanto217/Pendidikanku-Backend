// internals/features/lembaga/classes/sections/main/route/admin_route.go
package route

import (
	sectionctrl "madinahsalam_backend/internals/features/school/classes/class_sections/controller"
	schoolkuMiddleware "madinahsalam_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AllClassSectionRoutes(r fiber.Router, db *gorm.DB) {
	h := sectionctrl.NewClassSectionController(db)

	// ===== Base by school_id =====
	baseByID := r.Group("/:school_id")
	sectionsByID := baseByID.Group("/class-sections")
	sectionsByID.Get("/list", h.List)

	// ===== Base by school_slug =====
	baseBySlug := r.Group("/:school_slug",
		schoolkuMiddleware.UseSchoolScope(), // set ctx school dari slug
	)
	sectionsBySlug := baseBySlug.Group("/class-sections")
	sectionsBySlug.Get("/list", h.List)
}
