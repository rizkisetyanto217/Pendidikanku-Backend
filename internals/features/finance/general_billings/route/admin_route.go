// file: internals/features/finance/general_billings/route/general_billing_routes.go
package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	// Controller untuk GENERAL BILLINGS (finance/general_billings)
	gbController "masjidku_backend/internals/features/finance/general_billings/controller"
)

func AdminGeneralBillingRoutes(r fiber.Router, db *gorm.DB) {
	// ===== General Billings =====
	gbCtl := gbController.NewGeneralBillingController(db)
	gb := r.Group("/:masjid_id/general-billings")
	{
		gb.Post("/", gbCtl.Create)      // POST   /:masjid_id/general-billings
		gb.Patch("/:id", gbCtl.Patch)   // PATCH  /:masjid_id/general-billings/:id
		gb.Delete("/:id", gbCtl.Delete) // DELETE /:masjid_id/general-billings/:id
		gb.Get("/:id", gbCtl.GetByID)   // GET    /:masjid_id/general-billings/:id
		gb.Get("/", gbCtl.List)         // GET    /:masjid_id/general-billings
	}

	// ===== General Billing Kinds =====
	kindCtl := gbController.NewGeneralBillingKindController(db)
	kinds := r.Group("/:masjid_id/general-billing-kinds")
	{
		kinds.Post("/", kindCtl.Create)      // POST   /:masjid_id/general-billing-kinds
		kinds.Patch("/:id", kindCtl.Patch)   // PATCH  /:masjid_id/general-billing-kinds/:id
		kinds.Delete("/:id", kindCtl.Delete) // DELETE /:masjid_id/general-billing-kinds/:id
		// kinds.Get("/", kindCtl.List)       // (opsional jika sudah tersedia)
		// kinds.Get("/:id", kindCtl.GetByID) // (opsional jika sudah tersedia)
	}

	// ===== User General Billings =====
	ugbCtl := gbController.NewUserGeneralBillingController(db)
	ugb := r.Group("/:masjid_id/user-general-billings")
	{
		ugb.Post("/", ugbCtl.Create)      // POST   /:masjid_id/user-general-billings
		ugb.Patch("/:id", ugbCtl.Patch)   // PATCH  /:masjid_id/user-general-billings/:id
		ugb.Delete("/:id", ugbCtl.Delete) // DELETE /:masjid_id/user-general-billings/:id
		ugb.Get("/", ugbCtl.List)         // GET    /:masjid_id/user-general-billings
	}
}
