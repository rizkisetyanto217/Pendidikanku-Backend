// internals/routes/spp_billing_routes.go
package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	sppapi "schoolku_backend/internals/features/finance/billings/controller"
)

func BillingsUserRoutes(pub fiber.Router, db *gorm.DB) {
	// punya ListFeeRules, ListBillBatches, dst.
	studentBills := &sppapi.StudentBillHandler{DB: db} // punya List (atau ListStudentBills), Get, dll.
	// billBatches := &sppapi.BillBatchHandler{DB: db}

	grp := pub.Group("/:school_id")
	{
		// ---- Student Bills (readonly)
		// pakai StudentBillHandler agar tidak nyasar ke fee rule controller
		grp.Get("/student-bills/list", studentBills.List) // atau studentBills.ListStudentBills jika namanya begitu

	}
}
