// internals/features/lembaga/classes/sections/main/route/user_route.go
package route

import (
	sectionctrl "masjidku_backend/internals/features/school/classes/class_sections/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func ClassSectionUserRoutes(r fiber.Router, db *gorm.DB) {
	// Controllers
	sectionH := sectionctrl.NewClassSectionController(db)
	ucsH := sectionctrl.NewUserClassSectionController(db)

	// ================== PUBLIC (READ-ONLY) ==================
	pub := r.Group("/class-sections")
	// daftar section publik (support filter via query: term_id, grade, subject_id, teacher_id, q, page, size)
	pub.Get("/list", sectionH.ListClassSections)
	// pencarian cepat (q=keyword) – jika Anda ingin pisah dari list
	pub.Get("/search", sectionH.SearchClassSections)
	// detail by slug/id (untuk landing/SEO)
	pub.Get("/slug/:slug", sectionH.GetClassSectionBySlug)
	pub.Get("/:id", sectionH.GetClassSectionByID)
	// resource terkait (tidak mengekspos data sensitif user)
	pub.Get("/books/:id", sectionH.ListBooksBySection)

	// ================== USER (READ-ONLY) ==================
	user := r.Group("/user-class-sections")
	user.Get("/", ucsH.ListUserClassSections)
	user.Get("/:id", ucsH.GetUserClassSectionByID)
}
