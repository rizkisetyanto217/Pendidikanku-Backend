// file: internals/features/academics/terms/route/academic_term_admin_route.go
package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"masjidku_backend/internals/constants"
	academicTermCtl "masjidku_backend/internals/features/school/academics/academic_terms/controller"
	authMiddleware "masjidku_backend/internals/middlewares/auth"
	masjidkuMiddleware "masjidku_backend/internals/middlewares/features"
)

// ================================
// Admin/DKM routes (manage)
// Base group example: /api/a
// ================================
func AcademicYearAdminRoutes(api fiber.Router, db *gorm.DB) {
	// ================================
	// Academic Terms (CRUD + search)
	// => /api/a/academic-terms/...
	// ================================
	termCtl := academicTermCtl.NewAcademicTermController(db)

	adminTerms := api.Group("/academic-terms",
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola academic terms"),
			constants.AdminAndAbove,
		),
		masjidkuMiddleware.IsMasjidAdmin(),
	)

	adminTerms.Get("/list", termCtl.List)
	adminTerms.Get("/search", termCtl.SearchOnlyByYear) // taruh sebelum :id utk hindari bentrok
	adminTerms.Post("/", termCtl.Create)
	adminTerms.Put("/:id", termCtl.Update)
	adminTerms.Delete("/:id", termCtl.Delete)

}
