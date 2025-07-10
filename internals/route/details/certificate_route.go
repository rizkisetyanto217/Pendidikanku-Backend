package details

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func CertificateRoutes(app *fiber.App, db *gorm.DB) {
	// 🔐 Semua route aman (butuh token + rate limit)


}
