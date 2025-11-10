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
- /api/a/:school_id/payments ...
- /api/a/:school_slug/payments ...
*/
func PaymentAdminRoutes(r fiber.Router, db *gorm.DB, midtransServerKey string, useProd bool) {
	ctl := paymentController.NewPaymentController(db, midtransServerKey, useProd)

	// ====== BASE: by school_id ======
	baseByID := r.Group("/:school_id",
		schoolkuMiddleware.IsSchoolAdmin(), // guard DKM/admin
	)

	payByID := baseByID.Group("/payments")
	// LIST semua transaksi per tenant (sukses/gagal/pending)
	payByID.Get("/", ctl.ListPaymentsBySchoolAdmin)
	// CREATE payment (manual / gateway)
	payByID.Post("/", ctl.CreatePayment)
	// DETAIL + PATCH
	payByID.Patch("/:id", ctl.PatchPayment)

	// ====== BASE: by school_slug (opsional, kalau pakai slug/subdomain) ======
	baseBySlug := r.Group("/:school_slug",
		schoolkuMiddleware.UseSchoolScope(), // resolve slug -> school context (kalau kamu butuh)
		schoolkuMiddleware.IsSchoolAdmin(),
	)

	payBySlug := baseBySlug.Group("/payments")
	payBySlug.Get("/", ctl.ListPaymentsBySchoolAdmin)
	payBySlug.Post("/", ctl.CreatePayment)
	payBySlug.Patch("/:id", ctl.PatchPayment)
}
