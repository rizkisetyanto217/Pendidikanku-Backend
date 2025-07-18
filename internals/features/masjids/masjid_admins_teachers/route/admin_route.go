package route

import (
	"masjidku_backend/internals/features/masjids/masjid_admins_teachers/controller"
	masjidkuMiddleware "masjidku_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func MasjidAdminRoutes(api fiber.Router, db *gorm.DB) {
	ctrl := controller.NewMasjidAdminController(db)

	admin := api.Group("/masjid-admins")
	admin.Post("/", ctrl.AddAdmin)
	admin.Post("/by-masjid", ctrl.GetAdminsByMasjid)
	admin.Put("/revoke", ctrl.RevokeAdmin)

	ctrl2 := controller.NewMasjidTeacherController(db)

	teachers := api.Group("/masjid-teachers")
	teachers.Post("/",masjidkuMiddleware.IsMasjidAdmin(), ctrl2.Create)
	teachers.Get("/by-masjid", masjidkuMiddleware.IsMasjidAdmin(), ctrl2.GetByMasjid)
	teachers.Delete("/:id", masjidkuMiddleware.IsMasjidAdmin(), ctrl2.Delete)
}
