package route

import (
	"masjidku_backend/internals/features/lembaga/masjids_more/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// NOTE:
// - Taruh function ini di group /api/v1/public (opsional pakai SecondAuthMiddleware di parent).
// - Semua endpoint GET-only (read-only). CUD sudah dipindah ke admin/dkm routes.
func AllMasjidMoreRoutes(router fiber.Router, db *gorm.DB) {
	// 👤 Profil DKM & Teacher (public read)
	profileCtrl := controller.NewMasjidProfileTeacherDkmController(db)
	profile := router.Group("/masjid-profile-teacher-dkm")
	profile.Get("/",                 profileCtrl.GetProfilesByMasjid)     // ?masjid_id=...
	profile.Get("/:id",              profileCtrl.GetProfileByID)
	profile.Get("/by-masjid-slug/:slug", profileCtrl.GetProfilesByMasjidSlug)

	// 🏷️ Master Tag (public read)
	tagCtrl := controller.NewMasjidTagController(db)
	tag := router.Group("/masjid-tags")
	tag.Get("/", tagCtrl.GetAllTags)

	// 🔗 Relasi Tag ↔ Masjid (public read)
	tagRelCtrl := controller.NewMasjidTagRelationController(db)
	tagRel := router.Group("/masjid-tag-relations")
	tagRel.Get("/", tagRelCtrl.GetTagsByMasjid) // ?masjid_id=...

	// 📊 Statistik Masjid (public read)
	statsCtrl := controller.NewMasjidStatsController(db)
	stats := router.Group("/masjid-stats")
	stats.Get("/by-masjid", statsCtrl.GetStatsByMasjid) // ?masjid_id=...
}
