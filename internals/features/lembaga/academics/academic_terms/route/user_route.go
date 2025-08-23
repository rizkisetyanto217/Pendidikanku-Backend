// file: internals/features/academics/terms/route/academic_term_user_route.go
package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"masjidku_backend/internals/features/lembaga/academics/academic_terms/controller"
	classOpeningCtl "masjidku_backend/internals/features/lembaga/academics/academic_terms/controller"
	authMiddleware "masjidku_backend/internals/middlewares/auth"
)

// ================================
// User routes (read-only)
// Base path: /api/u/academic-year
// ================================
func AcademicYearUserRoutes(user fiber.Router, db *gorm.DB) {
	ctl := controller.NewAcademicTermController(db)
	openingCtl := classOpeningCtl.NewClassTermOpeningController(db)

	r := user.Group("/academic-terms",
		// pakai auth supaya helper.GetMasjidIDsFromToken(c) bekerja
		authMiddleware.AuthMiddleware(db),
	)

	// Read-only Academic Terms
	r.Get("/", ctl.List)
	r.Get("/:id", ctl.GetByID)

	// Search & distinct years (read-only juga)
	r.Get("/search", ctl.SearchByYear) // ?year=2026&page=1&page_size=20

	// =========================================
	// Class Term Openings (User - read only)
	// Base: /api/u/academic-terms/class-term-openings
	// =========================================
	open := r.Group("/class-term-openings")
	open.Get("/", openingCtl.GetAllClassTermOpenings) // dukung filter via query
	open.Get("/:id", openingCtl.GetClassTermOpeningByID)
}
