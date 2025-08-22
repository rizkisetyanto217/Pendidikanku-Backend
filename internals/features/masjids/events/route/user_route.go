package route

import (
	"masjidku_backend/internals/constants"
	"masjidku_backend/internals/features/masjids/events/controller"
	authMiddleware "masjidku_backend/internals/middlewares/auth"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Harus login; role apa pun yang diperbolehkan
func EventUserRoutes(api fiber.Router, db *gorm.DB) {
	user := api.Group("/",
		authMiddleware.AuthMiddleware(db),
		authMiddleware.OnlyRolesSlice(
			"❌ Hanya pengguna terautentikasi yang boleh mengakses fitur event user.",
			constants.AllowedRoles,
		),
	)

	// User Event Registrations (user daftar & lihat milik sendiri)
	regCtrl := controller.NewUserEventRegistrationController(db)
	reg := user.Group("/user-event-registrations")
	reg.Post("/", regCtrl.CreateRegistration)
	reg.Post("/by-user", regCtrl.GetRegistrantsByEvent)
}
