// file: internals/features/academics/terms/route/academic_term_user_route.go
package route

import (
	academicTermCtl "masjidku_backend/internals/features/school/academics/academic_terms/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// ================================
// User routes (read-only)
// Base group example: /api/u
// ================================
func AcademicYearUserRoutes(user fiber.Router, db *gorm.DB) {
	// ================================
	// Academic Terms (read-only)
	// => /api/u/academic-terms/...
	// ================================
	termCtl := academicTermCtl.NewAcademicTermController(db)

	r := user.Group("/academic-terms")

	// Read-only Academic Terms
	r.Get("/list", termCtl.List)
	r.Get("/search-with-class", termCtl.SearchByYear) // taruh sebelum :id agar tidak bentrok


}
