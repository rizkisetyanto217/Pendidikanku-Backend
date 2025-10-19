package route

// import (
// 	donationController "masjidku_backend/internals/features/finance/payments/controller"

// 	"github.com/gofiber/fiber/v2"
// 	"gorm.io/gorm"
// )

// // DonationRoutes defines the routes for donations
// func AllDonationRoutes(api fiber.Router, db *gorm.DB) {
// 	// Initialize donation controller
// 	donationCtrl := donationController.NewDonationController(db)
// 	api.Get("/midtrans/webhook", donationCtrl.MidtransWebhookPing) // ping

// 	// routes
// 	api.Post("/simple", donationCtrl.CreateDonationSimple) // /public/donations/simple

// 	// Define the donation routes
// 	api.Post("/:slug", donationCtrl.CreateDonation) // Create donation + Snap token
// 	api.Get("/", donationCtrl.GetAllDonations)      // Get all donations

// 	api.Get("/by-masjid/:slug", donationCtrl.GetDonationsByMasjidSlug)

// 	api.Get("/by-id/:id", donationCtrl.GetDonationByID)

// }
