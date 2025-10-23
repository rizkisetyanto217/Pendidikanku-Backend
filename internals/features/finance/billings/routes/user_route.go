// internals/routes/spp_billing_routes.go
package route

import (
	sppapi "masjidku_backend/internals/features/finance/billings/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

/*
Public routes (readonly)
GET list & detail untuk fee_rules, bill_batches, student_bills
*/
func BillingsPublicRoutes(pub fiber.Router, db *gorm.DB) {
	h := &sppapi.Handler{DB: db}

	// Grup memakai path masjid_id agar konsisten dengan modul lain
	grp := pub.Group("/:masjid_id")
	{
		// ---- Fee Rules (readonly)
		grp.Get("/fee-rules", h.ListFeeRules)

		// ---- Bill Batches (readonly)
		grp.Get("/bill-batches", h.ListFeeRules)

		// ---- Student Bills (readonly)
		grp.Get("/student-bills", h.ListStudentBills)
	}
}
