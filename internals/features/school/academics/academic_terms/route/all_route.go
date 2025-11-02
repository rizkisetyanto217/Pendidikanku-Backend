// file: internals/features/academics/terms/route/academic_term_user_route.go
package route

import (
	academicTermCtl "schoolku_backend/internals/features/school/academics/academic_terms/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// ================================
// User routes (read-only) — PUBLIC
// Base: /api/u/:school_id/academic-terms
// ================================
func AllAcademicTermsRoutes(user fiber.Router, db *gorm.DB) {
	termCtl := academicTermCtl.NewAcademicTermController(db, nil)

	// School context via PATH param :school_id
	r := user.Group("/:school_id/academic-terms")

	// Read-only Academic Terms
	r.Get("/list", termCtl.List)
	r.Get("/search-with-class", termCtl.SearchByYear) // taruh sebelum :id bila nanti ada detail
}
