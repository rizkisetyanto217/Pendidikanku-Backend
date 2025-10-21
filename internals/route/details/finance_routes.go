// file: internals/routes/details/finance_details_routes.go (misal)
package details

import (
	GeneralBillingRoute "masjidku_backend/internals/features/finance/general_billings/route"
	PaymentRoute        "masjidku_backend/internals/features/finance/payments/route"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func FinancePublicRoutes(r fiber.Router, db *gorm.DB) {
	GeneralBillingRoute.AllGeneralBillingRoutes(r, db)
	PaymentRoute.PaymentAllRoutes(r, db) // ← FIX: pakai r, bukan app
}

func FinanceAdminRoutes(r fiber.Router, db *gorm.DB) {
	GeneralBillingRoute.AdminGeneralBillingRoutes(r, db)
}
