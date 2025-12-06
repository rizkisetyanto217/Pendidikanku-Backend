// file: internals/routes/finance_payment_routes.go
package route

import (
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	paymentsController "madinahsalam_backend/internals/features/finance/payments/controller/payments"
)

func getenv(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func AllPaymentRoutes(r fiber.Router, db *gorm.DB) {
	midtransServerKey := getenv("MIDTRANS_SERVER_KEY", "")
	useProd := strings.EqualFold(getenv("MIDTRANS_USE_PROD", "false"), "true")

	h := paymentsController.NewPaymentController(db, midtransServerKey, useProd)

	// Base path di caller: /api/v1/finance
	payments := r.Group("/payments")

	// >>> INI dia endpoint eksplisit:
	payments.Post("/", h.CreatePayment)    // POST   /api/v1/finance/payments
	payments.Patch("/:id", h.PatchPayment) // PATCH  /api/v1/finance/payments/:id
}
