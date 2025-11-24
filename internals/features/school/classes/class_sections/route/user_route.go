// internals/features/lembaga/classes/sections/main/route/user_route.go
package route

import (
	sectionctrl "madinahsalam_backend/internals/features/school/classes/class_sections/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func ClassSectionUserRoutes(r fiber.Router, db *gorm.DB) {
	sectionH := sectionctrl.NewClassSectionController(db)
	ucsH := sectionctrl.NewStudentClassSectionController(db)

	r.Get("/class-sections/list", sectionH.List)

	user := r.Group("/student-class-sections")
	user.Post("/join", ucsH.JoinByCodeAutoSchool)
	user.Get("/list", ucsH.List) // ← pakai List baru
	user.Get("/detail/:id", ucsH.GetDetail)
	user.Post("/", ucsH.Create)
	user.Patch("/:id", ucsH.Patch)
	user.Delete("/:id", ucsH.Delete)
}
