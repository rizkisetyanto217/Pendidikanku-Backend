package route

import (
	"masjidku_backend/internals/features/masjids/masjids/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AllMasjidRoutes(user fiber.Router, db *gorm.DB) {
	masjidCtrl := controller.NewMasjidController(db)
	profileCtrl := controller.NewMasjidProfileController(db)

	// 🕌 Group: /masjids
	masjid := user.Group("/masjids")
	masjid.Get("/", masjidCtrl.GetAllMasjids)        // 📄 Semua masjid
	masjid.Get("/verified", masjidCtrl.GetAllVerifiedMasjids)
	masjid.Get("/:slug", masjidCtrl.GetMasjidBySlug) // 🔍 Detail by slug
	masjid.Get("/verified/:id", masjidCtrl.GetVerifiedMasjidByID)

	// 📄 Group: /masjid-profiles
	profile := user.Group("/masjid-profiles")
	profile.Get("/:masjid_id", profileCtrl.GetProfileByMasjidID) // 🔍 Profil masjid by masjid_id
	profile.Get("/by-slug/:slug", profileCtrl.GetProfileBySlug)


}
