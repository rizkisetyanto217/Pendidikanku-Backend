package route

import (
	"masjidku_backend/internals/features/masjids/masjids/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	masjidkuMiddleware "masjidku_backend/internals/middlewares/features"
)

func MasjidAdminRoutes(admin fiber.Router, db *gorm.DB) {
	masjidCtrl := controller.NewMasjidController(db)
	profileCtrl := controller.NewMasjidProfileController(db)

	// 🕌 Langsung gunakan admin.[METHOD] agar path param ":id" sudah terparse saat middleware dipanggil
	admin.Post("/masjids", masjidkuMiddleware.IsMasjidAdmin(), masjidCtrl.CreateMasjid)
	admin.Put("/masjids/:id", masjidkuMiddleware.IsMasjidAdmin(), masjidCtrl.UpdateMasjid)
	admin.Delete("/masjids/:id", masjidkuMiddleware.IsMasjidAdmin(), masjidCtrl.DeleteMasjid)

	admin.Post("/masjid-profiles", masjidkuMiddleware.IsMasjidAdmin(), profileCtrl.CreateMasjidProfile)
	admin.Put("/masjid-profiles/:id", masjidkuMiddleware.IsMasjidAdmin(), profileCtrl.UpdateMasjidProfile)
	admin.Delete("/masjid-profiles/:id", masjidkuMiddleware.IsMasjidAdmin(), profileCtrl.DeleteMasjidProfile)
}
