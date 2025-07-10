package route

import (
	"masjidku_backend/internals/features/masjids/masjid_admins/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func MasjidAdminRoutes(api fiber.Router, db *gorm.DB) {
	ctrl := controller.NewMasjidAdminController(db)

	admin := api.Group("/masjid-admins")
	admin.Post("/", ctrl.AddAdmin)
	admin.Post("/by-masjid", ctrl.GetAdminsByMasjid)
	admin.Put("/revoke", ctrl.RevokeAdmin)
}
