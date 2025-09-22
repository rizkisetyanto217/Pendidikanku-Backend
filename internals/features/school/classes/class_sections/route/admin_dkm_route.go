// internals/features/lembaga/classes/sections/main/route/admin_route.go
package route

import (
	"masjidku_backend/internals/constants"
	sectionctrl "masjidku_backend/internals/features/school/classes/class_sections/controller"
	authMiddleware "masjidku_backend/internals/middlewares/auth"
	masjidkuMiddleware "masjidku_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func ClassSectionAdminRoutes(api fiber.Router, db *gorm.DB) {
	// Controllers
	sectionH := sectionctrl.NewClassSectionController(db)
	ucsH := sectionctrl.NewUserClassSectionController(db)

	// Guard global: Admin/DKM + masjid admin check
	base := api.Group("",
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola class sections"),
			constants.AdminAndAbove,
		),
		masjidkuMiddleware.IsMasjidAdmin(),
	)

	// ========== 1) GENERIC (konteks via Header/Query/Host/Token) ==========
	base.Post("/class-sections", sectionH.CreateClassSection)
	base.Patch("/class-sections/:id", sectionH.UpdateClassSection)
	base.Delete("/class-sections/:id", sectionH.SoftDeleteClassSection)

	base.Post("/user-class-sections", ucsH.CreateUserClassSection)
	base.Patch("/user-class-sections/:id", ucsH.UpdateUserClassSection)
	base.Post("/user-class-sections/:id/end", ucsH.EndUserClassSection)
	base.Delete("/user-class-sections/:id", ucsH.DeleteUserClassSection)

	// ========== 2) PATH-SCOPED by masjid_id ==========
	base.Post("/:masjid_id/class-sections", sectionH.CreateClassSection)
	base.Patch("/:masjid_id/class-sections/:id", sectionH.UpdateClassSection)
	base.Delete("/:masjid_id/class-sections/:id", sectionH.SoftDeleteClassSection)

	base.Post("/:masjid_id/user-class-sections", ucsH.CreateUserClassSection)
	base.Patch("/:masjid_id/user-class-sections/:id", ucsH.UpdateUserClassSection)
	base.Post("/:masjid_id/user-class-sections/:id/end", ucsH.EndUserClassSection)
	base.Delete("/:masjid_id/user-class-sections/:id", ucsH.DeleteUserClassSection)

	// ========== 3) PATH-SCOPED by masjid_slug ==========
	base.Post("/m/:masjid_slug/class-sections", sectionH.CreateClassSection)
	base.Patch("/m/:masjid_slug/class-sections/:id", sectionH.UpdateClassSection)
	base.Delete("/m/:masjid_slug/class-sections/:id", sectionH.SoftDeleteClassSection)

	base.Post("/m/:masjid_slug/user-class-sections", ucsH.CreateUserClassSection)
	base.Patch("/m/:masjid_slug/user-class-sections/:id", ucsH.UpdateUserClassSection)
	base.Post("/m/:masjid_slug/user-class-sections/:id/end", ucsH.EndUserClassSection)
	base.Delete("/m/:masjid_slug/user-class-sections/:id", ucsH.DeleteUserClassSection)
}
