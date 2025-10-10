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

func AcademicTermsAdminRoutes(api fiber.Router, db *gorm.DB) {
	termCtl := academicTermCtl.NewAcademicTermController(db, nil)

	// Guard global (Admin/DKM + masjid admin check)
	base := api.Group("",
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola academic terms"),
			constants.AdminAndAbove,
		),
		masjidkuMiddleware.IsMasjidAdmin(),
	)

	// 1) GENERIC: konteks via Header/Query/Host (tetap didukung)
	base.Post("/academic-terms", termCtl.Create)
	base.Patch("/academic-terms/:id", termCtl.Patch)
	base.Delete("/academic-terms/:id", termCtl.Delete)

	// 2) PATH-SCOPED by masjid_id
	base.Post("/:masjid_id/academic-terms", termCtl.Create)
	base.Patch("/:masjid_id/academic-terms/:id", termCtl.Patch)
	base.Delete("/:masjid_id/academic-terms/:id", termCtl.Delete)

	// 3) PATH-SCOPED by masjid_slug
	base.Post("/m/:masjid_slug/academic-terms", termCtl.Create)
	base.Patch("/m/:masjid_slug/academic-terms/:id", termCtl.Patch)
	base.Delete("/m/:masjid_slug/academic-terms/:id", termCtl.Delete)
}
