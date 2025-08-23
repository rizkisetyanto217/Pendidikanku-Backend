package route

import (
	donationController "masjidku_backend/internals/features/payment/donations/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func DonationAdminRoutes(api fiber.Router, db *gorm.DB) {
	ctrl := donationController.NewDonationController(db)

	admin := api.Group("/donations")

	// 🔐 Hanya untuk admin masjid
	admin.Get("/by-masjid/:masjid_id", ctrl.GetDonationsByMasjidID) // ✅ ambil semua donasi completed untuk masjid_id
}