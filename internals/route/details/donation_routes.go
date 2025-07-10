package details

import (
	donationQuestionRoutes "masjidku_backend/internals/features/donations/donation_questions/route"
	donationController "masjidku_backend/internals/features/donations/donations/controller"
	donationRoutes "masjidku_backend/internals/features/donations/donations/routes"
	rateLimiter "masjidku_backend/internals/middlewares"
	authMiddleware "masjidku_backend/internals/middlewares/auth"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func DonationRoutes(app *fiber.App, db *gorm.DB) {
	// Semua route aman → membutuhkan token + rate limit
	api := app.Group("/api",
		authMiddleware.AuthMiddleware(db),
		rateLimiter.GlobalRateLimiter(),
	)

	// 👤 Route untuk user biasa (/api/u)
	userGroup := api.Group("/u")
	donationRoutes.DonationRoutes(userGroup, db) // data donasi user
	donationQuestionRoutes.DonationQuestionUserRoutes(userGroup.Group("/donation-questions"), db)

	// 🔐 Route untuk admin/owner (/api/a)
	adminGroup := api.Group("/a")
	donationQuestionRoutes.DonationQuestionAdminRoutes(adminGroup.Group("/donation-questions"), db)

	// 🔓 Webhook dari Midtrans (tidak pakai middleware)
	app.Post("/api/donations/notification", func(c *fiber.Ctx) error {
		c.Locals("db", db)
		return donationController.NewDonationController(db).HandleMidtransNotification(c)
	})
}
