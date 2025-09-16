// internals/features/masjid/service_plans/route/admin_service_plan_route.go
package route

import (
	"masjidku_backend/internals/constants"
	planctl "masjidku_backend/internals/features/lembaga/masjids/controller"
	"masjidku_backend/internals/middlewares/auth"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func MasjidOwnerRoutes(admin fiber.Router, db *gorm.DB) {
	guard := auth.OnlyRolesSlice(
		constants.RoleErrorAdmin("aksi ini untuk admin/owner"),
		constants.AdminAndAbove,
	)
	plan := planctl.NewMasjidServicePlanController(db)


	// alias lama (opsional):
	alias := admin.Group("/masjid-service-plans", guard)
	alias.Get("/",             plan.List)
	alias.Get("/:id",          plan.Detail)
	alias.Post("/",            plan.Create)
	alias.Patch("/:id",        plan.Update)
	alias.Delete("/:id",       plan.Delete)
	alias.Post("/:id/restore", plan.Restore)
}
