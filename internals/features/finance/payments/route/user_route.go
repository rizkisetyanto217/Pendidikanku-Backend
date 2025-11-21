// file: internals/routes/finance_payment_routes.go
package route

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	paymentctl "schoolku_backend/internals/features/finance/payments/controller"
)

func UserPaymentRoutes(r fiber.Router, db *gorm.DB) {
	midtransServerKey := getenv("MIDTRANS_SERVER_KEY", "")
	useProd := strings.EqualFold(getenv("MIDTRANS_USE_PROD", "false"), "true")

	h := paymentctl.NewPaymentController(db, midtransServerKey, useProd)

	// Semua user-level payments sekarang pakai school dari TOKEN,
	// bukan dari path. Prefix /payments saja.
	payments := r.Group("/payments")
	{
		// bundle registration + payment
		payments.Post("/registration-enroll", h.CreateRegistrationAndPayment)
		// kalau nanti mau expose generic create / patch untuk user,
		// bisa taruh di sini juga:
		// payments.Post("/", h.CreatePayment)
		// payments.Patch("/:id", h.PatchPayment)
	}
}
