package route

import (
	paymentController "schoolku_backend/internals/features/finance/payments/controller"
	schoolkuMiddleware "schoolku_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

/*
Admin routes: Payments (list + detail + create + patch)
Contoh mount: PaymentAdminRoutes(app.Group("/api/a"), db, midtransServerKey, useProd)
Final paths yang didukung:
- /api/a/payments ...
*/
func PaymentAdminRoutes(r fiber.Router, db *gorm.DB, midtransServerKey string, useProd bool) {
	ctl := paymentController.NewPaymentController(db, midtransServerKey, useProd)

	// BASE: payments by admin (school context diambil dari token/context)
	pay := r.Group("/payments",
		schoolkuMiddleware.IsSchoolAdmin(), // guard DKM/admin
	)

	// LIST semua transaksi per tenant (sukses/gagal/pending)
	pay.Get("/list", ctl.ListPaymentsBySchoolAdmin)
	// CREATE payment (manual / gateway)
	pay.Post("/", ctl.CreatePayment)
	// DETAIL + PATCH
	pay.Patch("/:id", ctl.PatchPayment)
}
