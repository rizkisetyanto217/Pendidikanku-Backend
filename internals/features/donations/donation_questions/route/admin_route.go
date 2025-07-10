package route

import (
	"masjidku_backend/internals/features/donations/donation_questions/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func DonationQuestionAdminRoutes(router fiber.Router, db *gorm.DB) {
	ctrl := controller.NewDonationQuestionController(db)

	router.Get("/", ctrl.GetAll)       // list semua
	router.Get("/:id", ctrl.GetByID)   // detail by id
	router.Post("/", ctrl.Create)      // tambah
	router.Put("/:id", ctrl.Update)    // update
	router.Delete("/:id", ctrl.Delete) // hapus
}
