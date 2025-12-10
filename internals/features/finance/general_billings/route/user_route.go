// file: internals/features/finance/general_billings/route/general_billing_routes.go
package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	// Controller untuk GENERAL BILLINGS (finance/general_billings)
	generalBillingController "madinahsalam_backend/internals/features/finance/general_billings/controller"
)

// AFTER (tanpa school_id / school_slug di path)
func GeneralBillingUserRoutes(r fiber.Router, db *gorm.DB) {
	// ===== General Billings (READ-ONLY) =====
	gbCtl := generalBillingController.NewGeneralBillingController(db)
	gb := r.Group("/general-billings")
	{
		// GET /general-billings/list
		gb.Get("/list", gbCtl.List)
	}

	// ===== User General Billings (READ-ONLY) =====
	ugbCtl := generalBillingController.NewUserGeneralBillingController(db)
	ugb := r.Group("/user-general-billings")
	{
		// GET /user-general-billings/list
		ugb.Get("/list", ugbCtl.List)
	}

}
