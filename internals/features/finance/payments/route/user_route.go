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
	// VARIAN A (direkomendasikan):
	// Caller base path: /api
	// Hasil: POST /api/a/:school_id/finance/payments/registration-enroll
	// ===========================
	payments := r.Group("/:school_id/payments")

	// ===========================
	// VARIAN B (kalau base path caller sudah /api/v1/finance)
	// Uncomment kalau kamu memanggil dari /api/v1/finance sebagai base:
	// payments := r.Group("/:school_id/payments")
	// ===========================

	// khusus student login: registrasi + payment + auto-enroll
	payments.Post("/registration-enroll", h.CreateRegistrationAndPayment)
}
