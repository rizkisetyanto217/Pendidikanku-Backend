package details

import (
	authRoute "masjidku_backend/internals/features/users/auth/route"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AuthRoutes(app *fiber.App, db *gorm.DB) {

	authRoute.AuthRoutes(app, db)

}
