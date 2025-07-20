package route

import (
	donationController "masjidku_backend/internals/features/donations/donations/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// func DonationRoutes(app *fiber.App, db *gorm.DB) {
// 	// Group semua route dengan rate limiter dan autentikasi
// 	api := app.Group("/api", auth.AuthMiddleware(db))

// 	// Group untuk user (/api/u)
// 	userGroup := api.Group("/u")
// 	// Donation routes untuk user
// 	donationRoutesForUser(userGroup, db)

// 	// Group untuk admin/owner (/api/a)
// 	adminGroup := api.Group("/a")
// 	// Donation question routes untuk admin
// 	donationQuestionRoutes.DonationQuestionAdminRoutes(adminGroup.Group("/donation-questions"), db)

// 	// Webhook dari Midtrans
// 	app.Post("/api/donations/notification", func(c *fiber.Ctx) error {
// 		c.Locals("db", db)
// 		return donationController.NewDonationController(db).HandleMidtransNotification(c)
// 	})
// }

// // Route untuk donasi user
// func donationRoutesForUser(api fiber.Router, db *gorm.DB) {
// 	donationCtrl := donationController.NewDonationController(db)

// 	// Group untuk donasi user
// 	donationRoutes := api.Group("/donations")
// 	// Route untuk donasi user
// 	donationRoutes.Post("/", donationCtrl.CreateDonation)                   // Buat donasi + Snap token
// 	donationRoutes.Get("/", donationCtrl.GetAllDonations)                   // Semua donasi
// 	donationRoutes.Get("/user/:user_id", donationCtrl.GetDonationsByUserID) // Donasi per user
// }

// DonationRoutes defines the routes for donations
func DonationRoutes(api fiber.Router, db *gorm.DB) {
	// Initialize donation controller
	donationCtrl := donationController.NewDonationController(db)

	// Define the donation routes
	api.Post("/", donationCtrl.CreateDonation)                   // Create donation + Snap token
	api.Get("/", donationCtrl.GetAllDonations)                   // Get all donations
	api.Get("/user/:user_id", donationCtrl.GetDonationsByUserID) // Get donations by user

	api.Get("/masjid/:slug", donationCtrl.GetDonationsByMasjidSlug)
	// Add route for Midtrans webhook to handle payment status updates
	api.Post("/midtrans/webhook", donationCtrl.HandleMidtransNotification) // Midtrans Webhook
}