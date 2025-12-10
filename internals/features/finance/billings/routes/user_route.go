// internals/routes/spp_billing_routes.go
package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	feeRulesController "madinahsalam_backend/internals/features/finance/billings/controller/fee_rules"
)

func BillingsUserRoutes(pub fiber.Router, db *gorm.DB) {
	// punya ListFeeRules, ListBillBatches, dst.
	feeRules := &feeRulesController.FeeRuleHandler{DB: db}

	grp := pub.Group("")
	{

		grp.Get("/fee-rules/list", feeRules.ListFeeRules)
	}
}
