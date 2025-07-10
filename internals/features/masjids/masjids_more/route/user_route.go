package route

import (
	"masjidku_backend/internals/features/masjids/masjids_more/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AllMasjidMoreRoutes(router fiber.Router, db *gorm.DB) {
	// 🎓 Endpoint untuk melihat daftar profil teacher/DKM (tanpa create/update/delete)
	ctrl := controller.NewMasjidProfileTeacherDkmController(db)
	profile := router.Group("/masjid-profile-teacher-dkm")
	profile.Post("/by-masjid", ctrl.GetProfilesByMasjid)

	// 🏷️ Endpoint untuk melihat tag yang tersedia
	ctrl2 := controller.NewMasjidTagController(db)
	tag := router.Group("/masjid-tags")
	tag.Get("/", ctrl2.GetAllTags)

	// 🔗 Endpoint untuk melihat relasi tag dan masjid
	ctrl3 := controller.NewMasjidTagRelationController(db)
	tagRelation := router.Group("/masjid-tag-relations")
	tagRelation.Get("/", ctrl3.GetTagsByMasjid)

	// 📊 Endpoint statistik masjid
	ctrl4 := controller.NewMasjidStatsController(db)
	stats := router.Group("/masjid-stats")
	stats.Get("/by-masjid", ctrl4.GetStatsByMasjid)
}
