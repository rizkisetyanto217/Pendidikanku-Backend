package route

import (
	donationController "masjidku_backend/internals/features/donations/donations/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)


// DonationRoutes defines the routes for donations
func AllDonationRoutes(api fiber.Router, db *gorm.DB) {
	// Initialize donation controller
	donationCtrl := donationController.NewDonationController(db)

	// Define the donation routes
	api.Post("/", donationCtrl.CreateDonation)                   // Create donation + Snap token
	api.Get("/", donationCtrl.GetAllDonations)                   // Get all donations
	api.Get("/user/:user_id", donationCtrl.GetDonationsByUserID) // Get donations by user

	api.Get("/by-masjid/:slug", donationCtrl.GetDonationsByMasjidSlug)

	api.Post("/midtrans/webhook", donationCtrl.HandleMidtransNotification) // Midtrans Webhook
}