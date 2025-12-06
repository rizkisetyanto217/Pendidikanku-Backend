// file: internals/routes/finance_payment_routes.go
package route

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	paymentsController "madinahsalam_backend/internals/features/finance/payments/controller/payments"
)

// file: internals/routes/finance_payment_routes.go
func UserPaymentRoutes(r fiber.Router, db *gorm.DB) {
	midtransServerKey := getenv("MIDTRANS_SERVER_KEY", "")
	useProd := strings.EqualFold(getenv("MIDTRANS_USE_PROD", "false"), "true")

	h := paymentsController.NewPaymentController(db, midtransServerKey, useProd)

	payments := r.Group("/payments")
	{
		payments.Post("/registration-enroll", h.CreateRegistrationAndPayment)

		// 🔹 LIST (default) + DETAIL (via query ?payment-id=...)
		payments.Get("/list", h.List)
	}
}
