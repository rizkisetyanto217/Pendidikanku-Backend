// file: internals/features/academics/terms/route/academic_term_user_route.go
package route

import (
	academicTermCtl "masjidku_backend/internals/features/school/academics/academic_terms/controller"
	authMiddleware "masjidku_backend/internals/middlewares/auth"

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

	r := user.Group("/academic-terms",
		// pakai auth supaya helper.GetMasjidIDsFromToken(c) bekerja
		authMiddleware.AuthMiddleware(db),
	)

	// Read-only Academic Terms
	r.Get("/", termCtl.List)
	r.Get("/search-with-class", termCtl.SearchByYear) // taruh sebelum :id agar tidak bentrok
	r.Get("/:id", termCtl.GetByID)

	// =========================================
	// Class Term Openings (standalone, read-only)
	// => /api/u/class-term-openings/...
	// =========================================
	openCtl := academicTermCtl.NewClassTermOpeningController(db)

	open := user.Group("/class-term-openings",
		authMiddleware.AuthMiddleware(db),
	)

	open.Get("/", openCtl.GetAllClassTermOpenings) // dukung filter via query
	open.Get("/:id", openCtl.GetClassTermOpeningByID)
}
