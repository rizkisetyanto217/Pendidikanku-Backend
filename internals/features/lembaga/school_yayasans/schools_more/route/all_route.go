package route

import (
	"madinahsalam_backend/internals/features/lembaga/school_yayasans/schools_more/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// NOTE:
// - Taruh function ini di group /api/v1/public (opsional pakai SecondAuthMiddleware di parent).
// - Semua endpoint GET-only (read-only). CUD sudah dipindah ke admin/dkm routes.
func AllSchoolMoreRoutes(router fiber.Router, db *gorm.DB) {
	// 👤 Profil DKM & Teacher (public read)
	profileCtrl := controller.NewSchoolProfileTeacherDkmController(db)
	profile := router.Group("/school-profile-teacher-dkm")
	profile.Get("/", profileCtrl.GetProfilesBySchool) // ?school_id=...
	profile.Get("/:id", profileCtrl.GetProfileByID)
	profile.Get("/by-school-slug/:slug", profileCtrl.GetProfilesBySchoolSlug)

	// 🏷️ Master Tag (public read)
	tagCtrl := controller.NewSchoolTagController(db)
	tag := router.Group("/school-tags")
	tag.Get("/", tagCtrl.GetAllTags)

	// 🔗 Relasi Tag ↔ School (public read)
	tagRelCtrl := controller.NewSchoolTagRelationController(db)
	tagRel := router.Group("/school-tag-relations")
	tagRel.Get("/", tagRelCtrl.GetTagsBySchool) // ?school_id=...

	// 📊 Statistik School (public read)
	statsCtrl := controller.NewSchoolStatsController(db)
	stats := router.Group("/school-stats")
	stats.Get("/by-school", statsCtrl.GetStatsBySchool) // ?school_id=...
}
