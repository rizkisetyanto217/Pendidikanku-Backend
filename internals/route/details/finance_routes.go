// file: internals/routes/details/finance_details_routes.go (misal)
package details

import (
	BillingRoute "masjidku_backend/internals/features/finance/billings/routes"
	GeneralBillingRoute "masjidku_backend/internals/features/finance/general_billings/route"
	PaymentRoute "masjidku_backend/internals/features/finance/payments/route"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func FinancePublicRoutes(r fiber.Router, db *gorm.DB) {
	GeneralBillingRoute.AllGeneralBillingRoutes(r, db)
	PaymentRoute.PaymentAllRoutes(r, db) // ← FIX: pakai r, bukan app
	BillingRoute.BillingsUserRoutes(r, db)

}

func FinanceAdminRoutes(r fiber.Router, db *gorm.DB) {
	GeneralBillingRoute.AdminGeneralBillingRoutes(r, db)

	BillingRoute.BillingsAdminRoutes(r, db)
}

func FinanceOwnerRoutes(r fiber.Router, db *gorm.DB) {
	GeneralBillingRoute.OwnerGeneralBillingRoutes(r, db)
}
