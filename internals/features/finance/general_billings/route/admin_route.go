// file: internals/features/finance/general_billings/route/general_billing_routes.go
package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	// Controller untuk GENERAL BILLINGS (finance/general_billings)
	gbController "masjidku_backend/internals/features/finance/general_billings/controller"
)

func AdminGeneralBillingRoutes(r fiber.Router, db *gorm.DB) {
	// ===== Tenant-scoped =====
	gbCtl := gbController.NewGeneralBillingController(db)
	gb := r.Group("/:masjid_id/general-billings")
	{
		gb.Post("/", gbCtl.Create)
		gb.Patch("/:id", gbCtl.Patch)
		gb.Delete("/:id", gbCtl.Delete)
	}

	kindCtl := gbController.NewGeneralBillingKindController(db)
	kinds := r.Group("/:masjid_id/general-billing-kinds")
	{
		kinds.Post("/", kindCtl.Create) // per-masjid
		kinds.Patch("/:id", kindCtl.Patch)
		kinds.Delete("/:id", kindCtl.Delete)
		// kinds.Get("/", kindCtl.ListTenant) // opsional; list per-masjid (+ include_global=true)
	}

	ugbCtl := gbController.NewUserGeneralBillingController(db)
	ugb := r.Group("/:masjid_id/user-general-billings")
	{
		ugb.Post("/", ugbCtl.Create)
		ugb.Patch("/:id", ugbCtl.Patch)
		ugb.Delete("/:id", ugbCtl.Delete)
		ugb.Get("/", ugbCtl.List)
	}

}
