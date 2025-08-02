package route

import (
	"masjidku_backend/internals/constants"
	"masjidku_backend/internals/features/masjids/masjids/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"masjidku_backend/internals/middlewares/auth"
	masjidkuMiddleware "masjidku_backend/internals/middlewares/features"
)

func MasjidAdminRoutes(admin fiber.Router, db *gorm.DB) {
	masjidCtrl := controller.NewMasjidController(db)
	profileCtrl := controller.NewMasjidProfileController(db)

	// 🕌 Langsung gunakan admin.[METHOD] agar path param ":id" sudah terparse saat middleware dipanggil
	admin.Post("/masjids",
	auth.OnlyRolesSlice(constants.RoleErrorOwner("membuat masjid"), constants.OwnerAndAbove),
	masjidCtrl.CreateMasjid,
)
	admin.Put("/masjids", masjidkuMiddleware.IsMasjidAdmin(), masjidCtrl.UpdateMasjid)
	admin.Delete("/masjids/:id", masjidkuMiddleware.IsMasjidAdmin(), masjidCtrl.DeleteMasjid)

	admin.Post("/masjid-profiles", masjidkuMiddleware.IsMasjidAdmin(), profileCtrl.CreateMasjidProfile)
	admin.Put("/masjid-profiles", masjidkuMiddleware.IsMasjidAdmin(), profileCtrl.UpdateMasjidProfile)
	admin.Delete("/masjid-profiles/:id", masjidkuMiddleware.IsMasjidAdmin(), profileCtrl.DeleteMasjidProfile)
}
