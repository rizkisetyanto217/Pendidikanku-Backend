// file: internals/features/finance/general_billings/route/general_billing_routes.go
package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	generalBillingController "madinahsalam_backend/internals/features/finance/general_billings/controller"
)

// AFTER
func AllGeneralBillingRoutes(r fiber.Router, db *gorm.DB) {
	// ===== General Billings (READ-ONLY) =====
	gbCtl := generalBillingController.NewGeneralBillingController(db)
	gb := r.Group("/:school_id/general-billings")
	{
		gb.Get("/list", gbCtl.List) // GET    /:school_id/general-billings
	}

	// ===== User General Billings (READ-ONLY) =====
	ugbCtl := generalBillingController.NewUserGeneralBillingController(db)
	ugb := r.Group("/:school_id/user-general-billings")
	{
		ugb.Get("/list", ugbCtl.List) // GET    /:school_id/user-general-billings
	}

}
