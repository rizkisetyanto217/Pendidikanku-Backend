// file: internals/features/finance/general_billings/route/general_billing_routes.go
package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	// Controller untuk GENERAL BILLINGS (finance/general_billings)

	// Controller untuk GENERAL BILLINGS (finance/general_billings)

	gbController "madinahsalam_backend/internals/features/finance/general_billings/controller"
	helperAuth "madinahsalam_backend/internals/helpers/auth"
)

func OwnerGeneralBillingRoutes(r fiber.Router, db *gorm.DB) {

	kindCtl := gbController.NewGeneralBillingKindController(db)

	// ===== GLOBAL admin (tanpa :school_id) untuk kinds =====
	admin := r.Group("/admin", helperAuth.OwnerOnly()) // <-- tambahkan middleware di sini

	adminKinds := admin.Group("/general-billing-kinds")
	{
		adminKinds.Post("/", kindCtl.CreateGlobal)
		adminKinds.Patch("/:id", kindCtl.PatchGlobal)
		adminKinds.Delete("/:id", kindCtl.DeleteGlobal)
	}
}
