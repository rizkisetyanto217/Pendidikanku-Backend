package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"masjidku_backend/internals/features/lembaga/academics/academic_year/controller"
)

// ================================
// User routes (read-only)
// Base path contoh: /api/u/academic-terms
// ================================
func AcademicTermUserRoutes(user fiber.Router, db *gorm.DB) {
	ctl := controller.NewAcademicTermController(db)

	r := user.Group("/academic-year")

	// read-only, tanpa create/update/delete
	r.Get("/", ctl.List)
	r.Get("/:id", ctl.GetByID)
}
