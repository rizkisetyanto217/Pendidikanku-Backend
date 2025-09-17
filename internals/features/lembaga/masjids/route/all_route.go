// file: internals/features/masjids/masjids/route/public_route.go (atau sesuai nama file kamu)
package route

import (
	"masjidku_backend/internals/features/lembaga/masjids/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AllMasjidRoutes(user fiber.Router, db *gorm.DB) {
	masjidCtrl  := controller.NewMasjidController(db, nil, nil)
	profileCtrl := controller.NewMasjidProfileController(db, nil)
	plan := controller.NewMasjidServicePlanController(db)

	// 🕌 Group: /masjids
	masjid := user.Group("/masjids")

	// Lebih spesifik dulu supaya tidak bentrok dengan "/:slug"
	masjid.Get("/verified",      masjidCtrl.GetAllVerifiedMasjids)
	masjid.Get("/verified/:id",  masjidCtrl.GetVerifiedMasjidByID)

	masjid.Get("/list",              masjidCtrl.GetAllMasjids)    // 📄 Semua masjid
	masjid.Get("/:slug",         masjidCtrl.GetMasjidBySlug)  // 🔍 Detail by slug

	// 📄 Group: /masjid-profiles
	profile := user.Group("/masjid-profiles")

	// Read-only endpoints yang tersedia di controller
	profile.Get("/list",                     profileCtrl.List)             // list + filter + pagination
	profile.Get("/nearest",              profileCtrl.Nearest)          // nearest?lat=&lon=&limit=



	// alias lama (opsional):
	alias := user.Group("/masjid-service-plans")
	alias.Get("/list",             plan.List)

}
