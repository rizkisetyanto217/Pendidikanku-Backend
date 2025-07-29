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

	api.Post("/midtrans/webhook", donationCtrl.HandleMidtransNotification) // Midtrans Webhook

	api.Get("/user/:user_id", donationCtrl.GetDonationsByUserID) // Get donations by user
	
	api.Get("/by-masjid/:slug", donationCtrl.GetDonationsByMasjidSlug)
	
	api.Get("/by-id/:id", donationCtrl.GetDonationByID)

	// ========== Donation Like Routes ==========
	donationLikeCtrl := donationController.NewDonationLikeController(db)

	api.Post("/likes/:slug/toggle", donationLikeCtrl.ToggleDonationLike)
	api.Get("/likes/count/:donation_id", donationLikeCtrl.GetDonationLikeCount)
	api.Get("/likes/is-liked/:donation_id", donationLikeCtrl.IsDonationLikedByUser)
	
}