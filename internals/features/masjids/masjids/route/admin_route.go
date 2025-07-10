package route

import (
	"masjidku_backend/internals/features/masjids/masjids/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func MasjidAdminRoutes(admin fiber.Router, db *gorm.DB) {
	masjidCtrl := controller.NewMasjidController(db)
	profileCtrl := controller.NewMasjidProfileController(db)

	// 🕌 Group: /masjids
	masjid := admin.Group("/masjids")
	masjid.Post("/", masjidCtrl.CreateMasjid)      // ➕ Buat masjid
	masjid.Put("/:id", masjidCtrl.UpdateMasjid)    // ✏️ Edit masjid
	masjid.Delete("/:id", masjidCtrl.DeleteMasjid) // ❌ Hapus masjid

	// 📄 Group: /masjid-profiles
	profile := admin.Group("/masjid-profiles")
	profile.Post("/", profileCtrl.CreateMasjidProfile)      // ➕ Buat profil masjid
	profile.Put("/:id", profileCtrl.UpdateMasjidProfile)    // ✏️ Edit profil masjid
	profile.Delete("/:id", profileCtrl.DeleteMasjidProfile) // ❌ Hapus profil
}
