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
	masjidURLCtrl  := controller.NewMasjidURLController(db, validator.New()) // << tambah controller URL

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

	// ==========================================
	// 🖼️ MASJID URLS (Admin/DKM) — KHUSUS FILE
	// ==========================================
	// Group khusus agar mudah tracking record per-file.
	// Biasanya tabel: masjid_urls (kolom umum: id, masjid_id, type, file_url, is_primary, is_active, order_index, created_at, ...)
	urls := admin.Group("/masjid-urls", guard)

	// List & detail
	urls.Get("/list", masjidURLCtrl.List)                                // GET    /api/a/masjid-urls?masjid_id=&type=&is_primary=&is_active=&q=&page=&page_size=
	urls.Get("/:id", masjidURLCtrl.GetByID)                          // GET    /api/a/masjid-urls/:id

	// Create / Update / Delete
	urls.Post("/", masjidURLCtrl.Create)                             // POST   /api/a/masjid-urls           (masjid_id, type, file_url, is_primary, is_active, order_index, label, ...)
	urls.Patch("/:id", masjidURLCtrl.Patch)                         // PATCH  /api/a/masjid-urls/:id
	urls.Delete("/:id", masjidURLCtrl.Delete)                        // DELETE /api/a/masjid-urls/:id
	// urls.Post("/:id/restore", masjidURLCtrl.Restore)                 // POST   /api/a/masjid-urls/:id/restore

	// Aksi khusus
	// urls.Post("/:id/set-primary", masjidURLCtrl.SetPrimary)          // POST   /api/a/masjid-urls/:id/set-primary  (opsional body: {"primary_type":"logo|cover|favicon|gallery"})
	// urls.Post("/reorder", masjidURLCtrl.Reorder)                     // POST   /api/a/masjid-urls/reorder         (body: [{"id":"...","order_index":1}, ...])

	// (Opsional) nested create/list by masjid, kalau mau pemakaian natural:
	// masjids.Post("/:id/urls", guard, masjidURLCtrl.CreateForMasjid) // POST /api/a/masjids/:id/urls
	// masjids.Get("/:id/urls/raw", guard, masjidURLCtrl.ListByMasjid) // GET  /api/a/masjids/:id/urls/raw (data mentah tabel masjid_urls)
}
