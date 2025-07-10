package route

import (
	"masjidku_backend/internals/features/masjids/masjids_more/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func MasjidMoreRoutes(router fiber.Router, db *gorm.DB) {
	ctrl := controller.NewMasjidProfileTeacherDkmController(db)

	profil_teacher_dkm := router.Group("/masjid-profile-teacher-dkm")
	profil_teacher_dkm.Post("/", ctrl.CreateProfile)
	profil_teacher_dkm.Put("/:id", ctrl.UpdateProfile)
	profil_teacher_dkm.Delete("/:id", ctrl.DeleteProfile)

	ctrl2 := controller.NewMasjidTagController(db)
	tag := router.Group("/masjid-tags")
	tag.Post("/", ctrl2.CreateTag)      // Tambah tag
	tag.Get("/", ctrl2.GetAllTags)      // Ambil semua tag
	tag.Delete("/:id", ctrl2.DeleteTag) // Hapus tag

	ctrl3 := controller.NewMasjidTagRelationController(db)
	tag_relation := router.Group("/masjid-tag-relations")
	tag_relation.Post("/", ctrl3.CreateRelation) // Tambah relasi tag ke masjid
	tag_relation.Get("/", ctrl3.GetTagsByMasjid) // Ambil semua tag yang terkait dengan masjid tertentu

	ctrl4 := controller.NewMasjidStatsController(db)
	stats := router.Group("/masjid-stats")
	stats.Post("/", ctrl4.UpsertStats)          // POST untuk tambah atau update data statistik
	stats.Get("/by-masjid", ctrl4.GetStatsByMasjid) // GET data statistik berdasarkan masjid_id (query param)
}