// internals/routes/spp_billing_routes.go
package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	feeRulesController "madinahsalam_backend/internals/features/finance/billings/controller/fee_rules"
)

func AllBillingRoutes(pub fiber.Router, db *gorm.DB) {
	feeAndBatch := &feeRulesController.FeeRuleHandler{DB: db} // punya ListFeeRules, ListBillBatches, dst.

	grp := pub.Group("/:school_id")
	{
		// ---- Fee Rules (readonly)
		grp.Get("/fee-rules/list", feeAndBatch.ListFeeRules)

	}
}
