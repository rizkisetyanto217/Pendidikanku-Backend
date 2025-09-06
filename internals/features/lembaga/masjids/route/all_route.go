// file: internals/features/masjids/masjids/route/public_route.go (atau sesuai nama file kamu)
package route

import (
	"masjidku_backend/internals/features/lembaga/masjids/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AllMasjidRoutes(user fiber.Router, db *gorm.DB) {
	masjidCtrl  := controller.NewMasjidController(db)
	profileCtrl := controller.NewMasjidProfileController(db, nil)

	// 🕌 Group: /masjids
	masjid := user.Group("/masjids")

	// Lebih spesifik dulu supaya tidak bentrok dengan "/:slug"
	masjid.Get("/verified",      masjidCtrl.GetAllVerifiedMasjids)
	masjid.Get("/verified/:id",  masjidCtrl.GetVerifiedMasjidByID)

	masjid.Get("/",              masjidCtrl.GetAllMasjids)    // 📄 Semua masjid
	masjid.Get("/:slug",         masjidCtrl.GetMasjidBySlug)  // 🔍 Detail by slug

	// 📄 Group: /masjid-profiles
	profile := user.Group("/masjid-profiles")

	// Read-only endpoints yang tersedia di controller
	profile.Get("/",                     profileCtrl.List)             // list + filter + pagination
	profile.Get("/nearest",              profileCtrl.Nearest)          // nearest?lat=&lon=&limit=
	profile.Get("/by-masjid/:masjid_id", profileCtrl.GetByMasjidID)    // profil by masjid_id (UUID)
	profile.Get("/:id",                  profileCtrl.GetByID)          // profil by profile_id (UUID)

	// Catatan: kamu sebelumnya pakai:
	//   profile.Get("/:masjid_id", profileCtrl.GetProfileByMasjidID)
	//   profile.Get("/by-slug/:slug", profileCtrl.GetProfileBySlug)
	// Handler tsb tidak ada di controller yang sekarang.
	// Jika memang butuh "by-slug", tambahkan handler GetBySlug di controller
	// yang join ke tabel masjids berdasarkan slug.
}
