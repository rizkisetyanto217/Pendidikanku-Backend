package route

import (
	"masjidku_backend/internals/constants"
	"masjidku_backend/internals/features/masjids/masjids/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"masjidku_backend/internals/middlewares/auth"
)

func MasjidAdminRoutes(admin fiber.Router, db *gorm.DB) {
	masjidCtrl := controller.NewMasjidController(db)
	profileCtrl := controller.NewMasjidProfileController(db)

	// =========================
	// Sub-group berdasar role
	// =========================

	// ✅ Hanya OWNER (super-admin) — aksi sensitif/lintas tenant
	ownerOnly := admin.Group("/",
		auth.OnlyRolesSlice(constants.RoleErrorOwner("aksi ini khusus owner"), constants.OwnerOnly),
	)

	// ✅ ADMIN (admin/dkm) & OWNER — operasi harian
	adminOrOwner := admin.Group("/",
		auth.OnlyRolesSlice(constants.RoleErrorAdmin("aksi ini untuk admin/owner"), constants.AdminAndAbove),
	)

	// =========================
	// 🕌 Masjid
	// =========================

	// Create masjid: OWNER only
	ownerOnly.Post("/masjids", masjidCtrl.CreateMasjid)

	// Update + delete: admin/dkm atau owner
	adminOrOwner.Put("/masjids", masjidCtrl.UpdateMasjid)
	adminOrOwner.Delete("/masjids", masjidCtrl.DeleteMasjid)      // by body
	adminOrOwner.Delete("/masjids/:id", masjidCtrl.DeleteMasjid)  // by param

	// =========================
	// 🏷️ Masjid Profile
	// =========================
	adminOrOwner.Post("/masjid-profiles",    profileCtrl.CreateMasjidProfile)
	adminOrOwner.Put("/masjid-profiles",     profileCtrl.UpdateMasjidProfile)
	adminOrOwner.Delete("/masjid-profiles",  profileCtrl.DeleteMasjidProfile)
	adminOrOwner.Delete("/masjid-profiles/:id", profileCtrl.DeleteMasjidProfile)
}
