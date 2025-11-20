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

	// ===========================
	// VARIAN ID (by UUID di path)
	// Contoh: POST /api/u/:school_id/finance/payments/registration-enroll
	// ===========================
	payments := r.Group("/:school_id/payments")
	{
		payments.Post("/registration-enroll", h.CreateRegistrationAndPayment)
	}

	// ===========================
	// VARIAN SLUG
	// Contoh: POST /api/u/m/:school_slug/finance/payments/registration-enroll
	// ===========================
	paymentsSlug := r.Group("/s/:school_slug/payments")
	{
		paymentsSlug.Post("/registration-enroll", h.CreateRegistrationAndPayment)
	}
}
