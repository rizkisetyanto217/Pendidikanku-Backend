// internals/features/lembaga/classes/sections/main/route/admin_route.go
package route

import (
	sectionctrl "masjidku_backend/internals/features/school/classes/class_sections/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AllClassSectionRoutes(r fiber.Router, db *gorm.DB) {
	h := sectionctrl.NewClassSectionController(db)

	// ===== Base by masjid_id =====
	baseByID := r.Group("/:masjid_id")
	sectionsByID := baseByID.Group("/class-sections")
	sectionsByID.Get("/list", h.ListClassSections)

}
