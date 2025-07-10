package route

import (
	"masjidku_backend/internals/features/donations/donation_questions/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func DonationQuestionUserRoutes(router fiber.Router, db *gorm.DB) {
	ctrl := controller.NewDonationQuestionController(db)

	router.Get("/:id", ctrl.GetByID)                          // user bisa lihat detail
	router.Get("/donation/:donationId", ctrl.GetByDonationID) // list dari donation tertentu
}
