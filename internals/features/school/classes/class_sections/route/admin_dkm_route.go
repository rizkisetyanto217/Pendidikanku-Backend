// internals/features/lembaga/classes/sections/main/route/admin_route.go
package route

import (
	sectionctrl "masjidku_backend/internals/features/school/classes/class_sections/controller"
	masjidkuMiddleware "masjidku_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func ClassSectionAdminRoutes(r fiber.Router, db *gorm.DB) {
	// Controllers
	sectionH := sectionctrl.NewClassSectionController(db)
	ucsH := sectionctrl.NewUserClassSectionController(db)

	// ================== CLASS SECTIONS ==================
	sections := r.Group("/class-sections", masjidkuMiddleware.IsMasjidAdmin())
	sections.Post("/", sectionH.CreateClassSection)
	sections.Get("/list", sectionH.ListClassSections)
	sections.Get("/slug/:slug", sectionH.GetClassSectionBySlug)
	sections.Put("/:id", sectionH.UpdateClassSection)
	sections.Delete("/:id", sectionH.SoftDeleteClassSection)

	// ================== USER CLASS SECTIONS ==================
	userClassSections := r.Group("/user-class-sections", masjidkuMiddleware.IsMasjidAdmin())
	userClassSections.Post("/", ucsH.CreateUserClassSection)
	userClassSections.Get("/list", ucsH.ListUserClassSections)
	userClassSections.Put("/:id", ucsH.UpdateUserClassSection)
	userClassSections.Post("/:id/end", ucsH.EndUserClassSection) // unassign/akhiri penempatan
	userClassSections.Delete("/:id", ucsH.DeleteUserClassSection)
}
