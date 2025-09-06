// file: internals/features/masjids/masjids/route/admin_dkm_route.go
package route

import (
	"masjidku_backend/internals/constants"
	"masjidku_backend/internals/features/lembaga/masjids/controller"
	"masjidku_backend/internals/middlewares/auth"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Tambahkan *validator.Validate agar bisa diteruskan ke controller
func MasjidAdminRoutes(admin fiber.Router, db *gorm.DB) {
	masjidCtrl  := controller.NewMasjidController(db)          // asumsi constructor ini tetap
	profileCtrl := controller.NewMasjidProfileController(db, nil) // << sesuai controller terbaru

	// =========================
	// 🕌 MASJID
	// =========================

	// Prefix: /masjids
	masjids := admin.Group("/masjids")

	// Admin/dkm/owner untuk operasi harian → /api/a/masjids/...
	masjidsAdmin := masjids.Group("/",
		auth.OnlyRolesSlice(constants.RoleErrorAdmin("aksi ini untuk admin/owner"), constants.AdminAndAbove),
	)
	masjidsAdmin.Put("/",       masjidCtrl.UpdateMasjid)
	masjidsAdmin.Delete("/",    masjidCtrl.DeleteMasjid)     // by body (kalau memang controller kamu dukung)
	masjidsAdmin.Delete("/:id", masjidCtrl.DeleteMasjid)     // by param

	// =========================
	// 🏷️ MASJID PROFILE
	// =========================

	// Prefix: /masjid-profiles
	profiles := admin.Group("/masjid-profiles",
		auth.OnlyRolesSlice(constants.RoleErrorAdmin("aksi ini untuk admin/owner"), constants.AdminAndAbove),
	)

	// Sesuaikan dengan handler yang ada di MasjidProfileController:
	// Create (POST /), Update (PATCH /:id), Delete (DELETE /:id)
	profiles.Post("/",        profileCtrl.Create)
	profiles.Patch("/:id",    profileCtrl.Update)
	profiles.Delete("/:id",   profileCtrl.Delete)
}
