package route

import (
	"schoolku_backend/internals/constants"
	"schoolku_backend/internals/features/schools/events/controller"
	authMiddleware "schoolku_backend/internals/middlewares/auth"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Harus login; role apa pun yang diperbolehkan
func EventUserRoutes(api fiber.Router, db *gorm.DB) {
	user := api.Group("/",
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
