// file: internals/features/academics/terms/route/academic_term_admin_route.go
package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"schoolku_backend/internals/constants"
	academicTermCtl "schoolku_backend/internals/features/school/academics/academic_terms/controller"
	authMiddleware "schoolku_backend/internals/middlewares/auth"
	schoolkuMiddleware "schoolku_backend/internals/middlewares/features"
)

func AcademicTermsAdminRoutes(api fiber.Router, db *gorm.DB) {
	termCtl := academicTermCtl.NewAcademicTermController(db, nil)

	// Guard global (Admin/DKM + school admin check)
	base := api.Group("",
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola academic terms"),
			constants.AdminAndAbove,
		),
		schoolkuMiddleware.IsSchoolAdmin(),
	)

	// 1) GENERIC: konteks via Header/Query/Host (tetap didukung)
	base.Post("/academic-terms", termCtl.Create)
	base.Patch("/academic-terms/:id", termCtl.Patch)
	base.Delete("/academic-terms/:id", termCtl.Delete)

	// // 2) PATH-SCOPED by school_id
	// base.Post("/:school_id/academic-terms", termCtl.Create)
	// base.Patch("/:school_id/academic-terms/:id", termCtl.Patch)
	// base.Delete("/:school_id/academic-terms/:id", termCtl.Delete)

	// // 3) PATH-SCOPED by school_slug
	// base.Post("/m/:school_slug/academic-terms", termCtl.Create)
	// base.Patch("/m/:school_slug/academic-terms/:id", termCtl.Patch)
	// base.Delete("/m/:school_slug/academic-terms/:id", termCtl.Delete)
}
