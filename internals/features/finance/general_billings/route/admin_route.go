// file: internals/features/finance/general_billings/route/general_billing_routes.go
package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	// Controller untuk GENERAL BILLINGS (finance/general_billings)
	generalBillingController "madinahsalam_backend/internals/features/finance/general_billings/controller/general"
	generalBillingKindController "madinahsalam_backend/internals/features/finance/general_billings/controller/kinds"
)

func AdminGeneralBillingRoutes(r fiber.Router, db *gorm.DB) {
	// ===== Tenant-scoped (school_id diambil dari token di controller) =====
	gbCtl := generalBillingController.NewGeneralBillingController(db)
	gb := r.Group("/general-billings")
	{
		gb.Post("/", gbCtl.Create)
		gb.Patch("/:id", gbCtl.Patch)
		gb.Delete("/:id", gbCtl.Delete)
	}

	kindCtl := generalBillingKindController.NewGeneralBillingKindController(db)
	kinds := r.Group("/general-billing-kinds")
	{
		// per-school, tapi school_id diambil dari token
		kinds.Post("/", kindCtl.Create)
		kinds.Patch("/:id", kindCtl.Patch)
		kinds.Delete("/:id", kindCtl.Delete)
	}

	ugbCtl := generalBillingController.NewUserGeneralBillingController(db)
	ugb := r.Group("/user-general-billings")
	{
		ugb.Post("/", ugbCtl.Create)
		ugb.Patch("/:id", ugbCtl.Patch)
		ugb.Delete("/:id", ugbCtl.Delete)
		ugb.Get("/", ugbCtl.List)
	}
}
