// internals/routes/spp_billing_routes.go
package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	sppapi "schoolku_backend/internals/features/finance/billings/controller"
)

func AllBillingRoutes(pub fiber.Router, db *gorm.DB) {
	feeAndBatch := &sppapi.Handler{DB: db} // punya ListFeeRules, ListBillBatches, dst.

	grp := pub.Group("/:school_id")
	{
		// ---- Fee Rules (readonly)
		grp.Get("/fee-rules/list", feeAndBatch.ListFeeRules)

	}
}
