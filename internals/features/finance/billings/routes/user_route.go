// internals/routes/spp_billing_routes.go
package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	sppapi "madinahsalam_backend/internals/features/finance/billings/controller"
)

func BillingsUserRoutes(pub fiber.Router, db *gorm.DB) {
	// punya ListFeeRules, ListBillBatches, dst.
	h := &sppapi.Handler{DB: db}
	studentBills := &sppapi.StudentBillHandler{DB: db} // punya List (atau ListStudentBills), Get, dll.
	// billBatches := &sppapi.BillBatchHandler{DB: db}

	grp := pub.Group("")
	{
		// ---- Student Bills (readonly)
		grp.Get("/student-bills/list", studentBills.List) // atau studentBills.ListStudentBills jika namanya begitu
		grp.Get("/fee-rules/list", h.ListFeeRules)
	}
}
