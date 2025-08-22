package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"masjidku_backend/internals/constants"
	"masjidku_backend/internals/features/lembaga/academics/academic_year/controller"
	authMiddleware "masjidku_backend/internals/middlewares/auth"
	masjidkuMiddleware "masjidku_backend/internals/middlewares/features"
)

// ================================
// Admin/DKM routes (manage)
// Base path contoh: /api/a/academic-terms
// ================================
func AcademicYearAdminRoutes(api fiber.Router, db *gorm.DB) {
	ctl := controller.NewAcademicTermController(db)

	admin := api.Group("/academic-year",
		authMiddleware.AuthMiddleware(db),
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola academic terms"),
			constants.AdminAndAbove,
		),
		masjidkuMiddleware.IsMasjidAdmin(),
	)

	admin.Get("/", ctl.List)
	admin.Get("/:id", ctl.GetByID)
	admin.Post("/", ctl.Create)
	admin.Put("/:id", ctl.Update)
	admin.Delete("/:id", ctl.Delete)
}
