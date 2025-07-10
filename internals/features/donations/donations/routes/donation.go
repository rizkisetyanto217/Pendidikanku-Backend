package route

import (
	"masjidku_backend/internals/features/donations/donations/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func DonationRoutes(api fiber.Router, db *gorm.DB) {
	donationCtrl := controller.NewDonationController(db)

	donationRoutes := api.Group("/donations")
	donationRoutes.Post("/", donationCtrl.CreateDonation)                   // Buat donasi + Snap token
	donationRoutes.Get("/", donationCtrl.GetAllDonations)                   // Semua donasi
	donationRoutes.Get("/user/:user_id", donationCtrl.GetDonationsByUserID) // Donasi per user

}
