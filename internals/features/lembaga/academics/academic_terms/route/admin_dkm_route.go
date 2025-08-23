// file: internals/features/academics/terms/route/academic_term_admin_route.go
package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"masjidku_backend/internals/constants"
	classOpeningCtl "masjidku_backend/internals/features/lembaga/academics/academic_terms/controller"
	authMiddleware "masjidku_backend/internals/middlewares/auth"
	masjidkuMiddleware "masjidku_backend/internals/middlewares/features"
)

// ================================
// Admin/DKM routes (manage)
// Base path: /api/a/academic-year
// ================================
func AcademicYearAdminRoutes(api fiber.Router, db *gorm.DB) {
	ctl := classOpeningCtl.NewAcademicTermController(db)
	openingCtl := classOpeningCtl.NewClassTermOpeningController(db)

	admin := api.Group("/academic-terms",
		authMiddleware.AuthMiddleware(db),
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola academic terms"),
			constants.AdminAndAbove,
		),
		masjidkuMiddleware.IsMasjidAdmin(),
	)

	// CRUD Academic Terms
	admin.Get("/", ctl.List)
	admin.Get("/:id", ctl.GetByID)
	admin.Post("/", ctl.Create)
	admin.Put("/:id", ctl.Update)
	admin.Delete("/:id", ctl.Delete)

	// Search & distinct years
	admin.Get("/search", ctl.SearchByYear) // ?year=2026&page=1&page_size=20

	// =========================================
	// Class Term Openings (Admin/DKM - manage)
	// Base: /api/a/academic-terms/class-term-openings
	// =========================================
	open := admin.Group("/class-term-openings")

	// List (with filters & pagination) — query: masjid_id, class_id, term_id, is_open, include_deleted, page, limit, sort
	open.Get("/", openingCtl.GetAllClassTermOpenings)
	// Detail
	open.Get("/:id", openingCtl.GetClassTermOpeningByID)
	// Create
	open.Post("/", openingCtl.CreateClassTermOpening)
	// Update
	open.Put("/:id", openingCtl.UpdateClassTermOpening)
	// Soft delete
	open.Delete("/:id", openingCtl.DeleteClassTermOpening)
	// Optional: restore soft-deleted
	open.Post("/:id/restore", openingCtl.RestoreClassTermOpening)
}
