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
		authMiddleware.AuthMiddleware(db),
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola academic terms"),
			constants.AdminAndAbove,
		),
		masjidkuMiddleware.IsMasjidAdmin(),
	)

	adminTerms.Get("/all", termCtl.List)
	adminTerms.Get("/search", termCtl.SearchOnlyByYear) // taruh sebelum :id utk hindari bentrok
	adminTerms.Get("/:id", termCtl.GetByID)
	adminTerms.Post("/", termCtl.Create)
	adminTerms.Put("/:id", termCtl.Update)
	adminTerms.Delete("/:id", termCtl.Delete)

	// =========================================
	// Class Term Openings (standalone, not nested)
	// => /api/a/class-term-openings/...
	// =========================================
	openingCtl := academicTermCtl.NewClassTermOpeningController(db)

	open := api.Group("/class-term-openings",
		authMiddleware.AuthMiddleware(db),
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola class term openings"),
			constants.AdminAndAbove,
		),
		masjidkuMiddleware.IsMasjidAdmin(),
	)

	// List (supports filters & pagination)
	open.Get("/all", openingCtl.GetAllClassTermOpenings)
	// Detail
	open.Get("/:id", openingCtl.GetClassTermOpeningByID)
	// Create
	open.Post("/", openingCtl.CreateClassTermOpening)
	// Update (partial)
	open.Put("/:id", openingCtl.UpdateClassTermOpening)
	// Soft delete
	open.Delete("/:id", openingCtl.DeleteClassTermOpening)
	// Optional: restore soft-deleted
	open.Post("/:id/restore", openingCtl.RestoreClassTermOpening)
}
