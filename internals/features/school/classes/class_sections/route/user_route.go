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
	// pub.Get("/slug/:slug", sectionH.GetClassSectionBySlug) // kalau perlu

	// ================== USER (scoped by masjid_id in path) ==================
	user := admin.Group("/:masjid_id/user-class-sections")
	user.Get("/list", ucsH.ListMine)        // optional list milik user
	user.Get("/detail/:id", ucsH.GetDetail) // detail by id
	user.Post("/", ucsH.Create)             // create
	user.Patch("/:id", ucsH.Patch)          // patch
	user.Delete("/:id", ucsH.Delete)        // soft delete
	user.Post("/join", ucsH.JoinByCode)     // join by code (kalau sudah ada)
}
