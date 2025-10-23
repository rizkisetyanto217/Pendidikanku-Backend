// file: internals/features/finance/general_billings/route/general_billing_routes.go
package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	// General Billings controller (create/patch/delete/get/list)
	gbController "masjidku_backend/internals/features/finance/general_billings/controller"
)

// AFTER
func AllGeneralBillingRoutes(r fiber.Router, db *gorm.DB) {
	// ===== General Billings (READ-ONLY) =====
	gbCtl := gbController.NewGeneralBillingController(db)
	gb := r.Group("/:masjid_id/general-billings")
	{
		gb.Get("/list", gbCtl.List)       // GET    /:masjid_id/general-billings
	}

	// ===== General Billing Kinds (READ-ONLY per snippet) =====
	kindCtl := gbController.NewGeneralBillingKindController(db)
	kinds := r.Group("/:masjid_id/general-billing-kinds")
	{
		kinds.Get("/list", kindCtl.List) // GET /:masjid_id/general-billing-kinds/list
	}

	// ===== User General Billings (READ-ONLY) =====
	ugbCtl := gbController.NewUserGeneralBillingController(db)
	ugb := r.Group("/:masjid_id/user-general-billings")
	{
		ugb.Get("/list", ugbCtl.List) // GET    /:masjid_id/user-general-billings
	}

	// ===== PUBLIC read-only (opsional) =====
	public := r.Group("/public")
	{
		publicKinds := public.Group("/general-billing-kinds")
		publicKinds.Get("/", kindCtl.ListPublic)
		publicKinds.Get("/:id", kindCtl.GetPublicByID)
	}
}
