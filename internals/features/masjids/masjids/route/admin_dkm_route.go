// file: internals/features/masjids/masjids/route/admin_dkm_route.go
package route

import (
	"masjidku_backend/internals/constants"
	"masjidku_backend/internals/features/masjids/masjids/controller"
	"masjidku_backend/internals/middlewares/auth"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func MasjidAdminRoutes(admin fiber.Router, db *gorm.DB) {
	masjidCtrl  := controller.NewMasjidController(db)
	profileCtrl := controller.NewMasjidProfileController(db)

	// =========================
	// 🕌 MASJID
	// =========================

	// Prefix: /masjids
	masjids := admin.Group("/masjids")

	// OWNER-only untuk aksi sensitif/lintas tenant → /api/a/masjids/owner/...
	masjidsOwner := masjids.Group("/owner",
		auth.OnlyRolesSlice(constants.RoleErrorOwner("aksi ini khusus owner"), constants.OwnerOnly),
	)
	masjidsOwner.Post("/", masjidCtrl.CreateMasjid)

	// Admin/dkm/owner untuk operasi harian → /api/a/masjids/...
	masjidsAdmin := masjids.Group("/",
		auth.OnlyRolesSlice(constants.RoleErrorAdmin("aksi ini untuk admin/owner"), constants.AdminAndAbove),
	)
	masjidsAdmin.Put("/",      masjidCtrl.UpdateMasjid)
	masjidsAdmin.Delete("/",   masjidCtrl.DeleteMasjid)     // by body
	masjidsAdmin.Delete("/:id", masjidCtrl.DeleteMasjid)    // by param

	// =========================
	// 🏷️ MASJID PROFILE
	// =========================

	// Prefix: /masjid-profiles
	profiles := admin.Group("/masjid-profiles",
		auth.OnlyRolesSlice(constants.RoleErrorAdmin("aksi ini untuk admin/owner"), constants.AdminAndAbove),
	)
	profiles.Post("/",        profileCtrl.CreateMasjidProfile)
	profiles.Put("/",         profileCtrl.UpdateMasjidProfile)
	profiles.Delete("/",      profileCtrl.DeleteMasjidProfile)
	profiles.Delete("/:id",   profileCtrl.DeleteMasjidProfile)
}
