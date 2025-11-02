// internals/features/lembaga/classes/sections/main/route/user_route.go
package route

import (
	sectionctrl "schoolku_backend/internals/features/school/classes/class_sections/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func ClassSectionUserRoutes(admin fiber.Router, db *gorm.DB) {
	sectionH := sectionctrl.NewClassSectionController(db)
	ucsH := sectionctrl.NewStudentClassSectionController(db)

	// ================== PUBLIC (READ-ONLY) ==================
	pub := admin.Group("/:school_id/class-sections")
	pub.Get("/list", sectionH.ListClassSections)

	// ================== USER (scoped by school_id in path) ==================
	user := admin.Group("/:school_id/student-class-sections")
	user.Get("/me", ucsH.ListMine)
	user.Get("/detail/:id", ucsH.GetDetail)
	user.Post("/", ucsH.Create)
	user.Patch("/:id", ucsH.Patch)
	user.Delete("/:id", ucsH.Delete)

}
