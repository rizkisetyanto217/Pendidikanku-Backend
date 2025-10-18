// internals/features/lembaga/classes/sections/main/route/user_route.go
package route

import (
	sectionctrl "masjidku_backend/internals/features/school/classes/class_sections/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)



func ClassSectionUserRoutes(admin fiber.Router, db *gorm.DB) {
	sectionH := sectionctrl.NewClassSectionController(db)
	ucsH := sectionctrl.NewUserClassSectionController(db)

	// ================== PUBLIC (READ-ONLY) ==================
	pub := admin.Group("/:masjid_id/class-sections")
	pub.Get("/list", sectionH.ListClassSections)

	// ================== USER (scoped by masjid_id in path) ==================
	user := admin.Group("/:masjid_id/user-class-sections")
	user.Get("/list", ucsH.ListMine)
	user.Get("/detail/:id", ucsH.GetDetail)
	user.Post("/", ucsH.Create)
	user.Patch("/:id", ucsH.Patch)
	user.Delete("/:id", ucsH.Delete)


}
