package route

import (
	donationController "masjidku_backend/internals/features/payment/donations/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// DonationRoutes defines the routes for donations
func AllDonationRoutes(api fiber.Router, db *gorm.DB) {
	// Initialize donation controller
	donationCtrl := donationController.NewDonationController(db)

	// Define the donation routes
	api.Post("/:slug", donationCtrl.CreateDonation)                   // Create donation + Snap token
	api.Get("/", donationCtrl.GetAllDonations)                   // Get all donations

	api.Post("/midtrans/webhook", donationCtrl.HandleMidtransNotification) // Midtrans Webhook

	api.Get("/by-user/:slug", donationCtrl.GetDonationsByUserIDWithSlug) // Get donations by user
	
	api.Get("/by-masjid/:slug", donationCtrl.GetDonationsByMasjidSlug)
	
	api.Get("/by-id/:id", donationCtrl.GetDonationByID)

	
}