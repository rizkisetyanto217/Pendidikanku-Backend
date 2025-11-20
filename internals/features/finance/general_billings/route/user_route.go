// file: internals/features/finance/general_billings/route/general_billing_routes.go
package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	// General Billings controller (create/patch/delete/get/list)
	gbController "schoolku_backend/internals/features/finance/general_billings/controller"
)

// AFTER (tanpa school_id / school_slug di path)
func GeneralBillingUserRoutes(r fiber.Router, db *gorm.DB) {
	// ===== General Billings (READ-ONLY) =====
	gbCtl := gbController.NewGeneralBillingController(db)
	gb := r.Group("/general-billings")
	{
		// GET /general-billings/list
		gb.Get("/list", gbCtl.List)
	}

	// ===== General Billing Kinds (READ-ONLY per tenant) =====
	kindCtl := gbController.NewGeneralBillingKindController(db)
	kinds := r.Group("/general-billing-kinds")
	{
		// GET /general-billing-kinds/list
		kinds.Get("/list", kindCtl.List)
	}

	// ===== General Billing Kinds (APP/GLOBAL) =====
	kindsApp := r.Group("/general-billing-kinds")
	{
		// tetap bisa dipakai sebagai list tenant-scope
		kindsApp.Get("/list", kindCtl.List) // GET /general-billing-kinds/list
		// list global untuk app (tanpa filter tenant)
		kindsApp.Get("/list-app", kindCtl.ListGlobal)
	}

	// ===== User General Billings (READ-ONLY) =====
	ugbCtl := gbController.NewUserGeneralBillingController(db)
	ugb := r.Group("/user-general-billings")
	{
		// GET /user-general-billings/list
		ugb.Get("/list", ugbCtl.List)
	}

	// ===== PUBLIC read-only (opsional) =====
	public := r.Group("/public")
	{
		publicKinds := public.Group("/general-billing-kinds")
		publicKinds.Get("/", kindCtl.ListPublic)       // GET /public/general-billing-kinds
		publicKinds.Get("/:id", kindCtl.GetPublicByID) // GET /public/general-billing-kinds/:id
	}
}
