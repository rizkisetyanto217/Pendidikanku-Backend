// file: internals/features/academics/terms/route/academic_term_user_route.go
package route

import (
	academicTermCtl "madinahsalam_backend/internals/features/school/academics/academic_terms/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// ===========================================
// User routes (read-only) — PUBLIC
//
// Pola yang didukung di sini:
//
//   - GET /api/u/:school_id/academic-terms/list
//   - GET /api/u/m/:school_slug/academic-terms/list
//
// Versi yang pakai token (tanpa :school_id / :school_slug) bisa
// kamu buat terpisah, misalnya di UserAcademicTermsRoutes.
// ===========================================
func AllAcademicTermsRoutes(user fiber.Router, db *gorm.DB) {
	termCtl := academicTermCtl.NewAcademicTermController(db, nil)

	// 1) /api/u/:school_id/academic-terms/list
	rByID := user.Group("/i/:school_id/academic-terms")
	rByID.Get("/list", termCtl.List)

	// 2) /api/u/m/:school_slug/academic-terms/list
	rBySlug := user.Group("/s/:school_slug/academic-terms")
	rBySlug.Get("/list", termCtl.List)
}
