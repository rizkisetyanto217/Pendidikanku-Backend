// file: internals/features/masjids/masjids/route/admin_dkm_route.go
package route

import (
	"masjidku_backend/internals/constants"
	"masjidku_backend/internals/middlewares/auth"

	masjidctl "masjidku_backend/internals/features/lembaga/masjids/controller"


	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Registrasi semua route Admin/DKM untuk fitur Masjid
func MasjidAdminRoutes(admin fiber.Router, db *gorm.DB) {
	// controllers
	masjidCtrl  := masjidctl.NewMasjidController(db, validator.New(), nil)
	profileCtrl := masjidctl.NewMasjidProfileController(db, validator.New())
	planCtrl    := masjidctl.NewMasjidServicePlanController(db) // ✅ dari paket yg benar

	// guard admin/dkm/owner
	guard := auth.OnlyRolesSlice(
		constants.RoleErrorAdmin("aksi ini untuk admin/owner"),
		constants.AdminAndAbove,
	)

	// =========================
	// 🕌 MASJID (Admin/DKM)
	// =========================
	masjids := admin.Group("/masjids")
	masjids.Get("/", guard, masjidCtrl.GetMasjids)
	masjids.Get("/:id_or_slug", guard, masjidCtrl.GetMasjid)
	masjids.Get("/:id/profile", guard, masjidCtrl.GetMasjidProfile)
	masjids.Get("/:id/urls", guard, masjidCtrl.GetMasjidURLs)

	masjids.Post("/", guard, masjidCtrl.CreateMasjidDKM)
	masjids.Put("/:id", guard, masjidCtrl.Patch)
	masjids.Delete("/:id", guard, masjidCtrl.Delete)

	// ================================
	// 🏷️ MASJID PROFILE (Admin/DKM)
	// ================================
	profiles := admin.Group("/masjid-profiles", guard)
	profiles.Post("/",      profileCtrl.Create)
	profiles.Patch("/:id",  profileCtrl.Update)
	profiles.Delete("/:id", profileCtrl.Delete)

	// ===================================
	// 🧩 SERVICE PLANS (Admin/Owner) — GLOBAL (tanpa MASJID_CTX)
	// ===================================
	// Alias kompat lama:
	alias := admin.Group("/masjid-service-plans", guard)
	alias.Get("/",             planCtrl.List)
	alias.Post("/",            planCtrl.Create)
	alias.Patch("/:id",        planCtrl.Patch)
	alias.Delete("/:id",       planCtrl.SoftDelete)
}
