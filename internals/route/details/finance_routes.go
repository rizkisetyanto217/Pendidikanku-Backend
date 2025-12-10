// file: internals/routes/details/finance_details_routes.go
package details

import (
	BillingRoute "madinahsalam_backend/internals/features/finance/billings/routes"
	GeneralBillingRoute "madinahsalam_backend/internals/features/finance/general_billings/route"
	PaymentRoute "madinahsalam_backend/internals/features/finance/payments/route" // ⬅️ pastikan paketnya "router"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func FinancePublicRoutes(r fiber.Router, db *gorm.DB) {
	GeneralBillingRoute.AllGeneralBillingRoutes(r, db)
	PaymentRoute.AllPaymentRoutes(r, db)
	BillingRoute.AllBillingRoutes(r, db)
}

func FinanceAdminRoutes(r fiber.Router, db *gorm.DB, midtransServerKey string, useProd bool) {
	GeneralBillingRoute.AdminGeneralBillingRoutes(r, db)
	PaymentRoute.PaymentAdminRoutes(r, db, midtransServerKey, useProd) // ✅ pass 4 args
	BillingRoute.BillingsAdminRoutes(r, db)
}

func FinanceUserRoutes(r fiber.Router, db *gorm.DB) {
	PaymentRoute.UserPaymentRoutes(r, db)
	BillingRoute.BillingsUserRoutes(r, db)
	GeneralBillingRoute.GeneralBillingUserRoutes(r, db)

}
