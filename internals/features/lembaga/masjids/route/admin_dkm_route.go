// file: internals/features/masjids/masjids/route/admin_dkm_route.go
package route

import (
	"masjidku_backend/internals/constants"
	"masjidku_backend/internals/features/lembaga/masjids/controller"
	"masjidku_backend/internals/middlewares/auth"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Registrasi semua route Admin/DKM untuk fitur Masjid
func MasjidAdminRoutes(admin fiber.Router, db *gorm.DB) {
	// controllers
	masjidCtrl     := controller.NewMasjidController(db)
	profileCtrl    := controller.NewMasjidProfileController(db, validator.New())

	// guard admin/dkm/owner
	guard := auth.OnlyRolesSlice(
		constants.RoleErrorAdmin("aksi ini untuk admin/owner"),
		constants.AdminAndAbove,
	)

	// =========================
	// 🕌 MASJID (Admin/DKM)
	// =========================
	masjids := admin.Group("/masjids")

	// READ (admin-only) – kalau ingin public, pindahkan ke public routes
	masjids.Get("/", guard, masjidCtrl.GetMasjids)                   // GET    /api/a/masjids
	masjids.Get("/:id_or_slug", guard, masjidCtrl.GetMasjid)         // GET    /api/a/masjids/:id_or_slug
	masjids.Get("/:id/profile", guard, masjidCtrl.GetMasjidProfile)  // GET    /api/a/masjids/:id/profile
	masjids.Get("/:id/urls", guard, masjidCtrl.GetMasjidURLs)        // GET    /api/a/masjids/:id/urls (view gabungan)

	// WRITE
	masjids.Post("/", guard, masjidCtrl.CreateMasjidDKM)             // POST   /api/a/masjids
	masjids.Put("/", guard, masjidCtrl.UpdateMasjid)                 // PUT    /api/a/masjids    (by body)
	masjids.Delete("/", guard, masjidCtrl.DeleteMasjid)              // DELETE /api/a/masjids    (by body)
	masjids.Delete("/:id", guard, masjidCtrl.DeleteMasjid)           // DELETE /api/a/masjids/:id

	// ================================
	// 🏷️ MASJID PROFILE (Admin/DKM)
	// ================================
	profiles := admin.Group("/masjid-profiles", guard)
	profiles.Post("/",       profileCtrl.Create)                     // POST   /api/a/masjid-profiles
	profiles.Patch("/:id",   profileCtrl.Update)                     // PATCH  /api/a/masjid-profiles/:id
	profiles.Delete("/:id",  profileCtrl.Delete)                     // DELETE /api/a/masjid-profiles/:id


}
