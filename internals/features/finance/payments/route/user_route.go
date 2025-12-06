// file: internals/routes/finance_payment_routes.go
package route

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	paymentsController "madinahsalam_backend/internals/features/finance/payments/controller/payments"

	paymentItemsController "madinahsalam_backend/internals/features/finance/payments/controller/items"
)

func UserPaymentRoutes(r fiber.Router, db *gorm.DB) {
	midtransServerKey := getenv("MIDTRANS_SERVER_KEY", "")
	useProd := strings.EqualFold(getenv("MIDTRANS_USE_PROD", "false"), "true")

	h := paymentsController.NewPaymentController(db, midtransServerKey, useProd)
	itemHandler := paymentItemsController.NewPaymentItemController(db)

	payments := r.Group("/payments")
	{
		payments.Post("/registration-enroll", h.CreateRegistrationAndPayment)

		// list header payment
		payments.Get("/list", h.List)

		// 🔹 list payment_items (flexibel)
		// - GET /payments/items               → semua item di school (role staff)
		// - GET /payments/items?student_id=me → item milik murid di token
		// - GET /payments/items?payment_id=... (&student_id=me optional)
	}

	paymentItems := r.Group("/payment-items")
	{
		paymentItems.Get("/list", itemHandler.ListPaymentItems)

	}
}
