// internals/features/lembaga/classes/sections/main/route/user_route.go
package route

import (
	sectionctrl "schoolku_backend/internals/features/school/classes/class_sections/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func ClassSectionUserRoutes(r fiber.Router, db *gorm.DB) {
	sectionH := sectionctrl.NewClassSectionController(db)
	ucsH := sectionctrl.NewStudentClassSectionController(db)

	// ================== PUBLIC / TOKEN-BASED (READ-ONLY) ==================
	// List class sections untuk school yang ter-resolve dari:
	// - token (preferred, via ResolveSchoolContext)
	// - atau query ?school_id / ?school_slug jika tidak ada token
	pub := r.Group("/class-sections")
	pub.Get("/list", sectionH.List)

	// ================== USER (pakai school dari token / context) ==================
	user := r.Group("/student-class-sections")

	// Self enrollment via join code (auto resolve school dari code section)
	// POST /api/u/student-class-sections/join
	user.Post("/join", ucsH.JoinByCodeAutoSchool)

	// CRUD student-class-sections scoped user/school
	user.Get("/me", ucsH.ListMine)
	user.Get("/detail/:id", ucsH.GetDetail)
	user.Post("/", ucsH.Create)
	user.Patch("/:id", ucsH.Patch)
	user.Delete("/:id", ucsH.Delete)
}
