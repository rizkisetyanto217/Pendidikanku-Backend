// internals/features/school/service_plans/route/admin_service_plan_route.go
package route

import (
	"schoolku_backend/internals/constants"
	planctl "schoolku_backend/internals/features/lembaga/school_yayasans/schools/controller"
	"schoolku_backend/internals/middlewares/auth"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func SchoolOwnerRoutes(admin fiber.Router, db *gorm.DB) {
	guard := auth.OnlyRolesSlice(
		constants.RoleErrorAdmin("aksi ini untuk admin/owner"),
		constants.AdminAndAbove,
	)
	plan := planctl.NewSchoolServicePlanController(db, nil)

	// alias lama (opsional):
	alias := admin.Group("/school-service-plans", guard)
	alias.Post("/", plan.Create)
	alias.Patch("/:id", plan.Patch)
	alias.Delete("/:id", plan.Delete)
}
