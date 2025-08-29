package route

import (
	"masjidku_backend/internals/constants"
	"masjidku_backend/internals/features/lembaga/masjids_more/controller"
	authMiddleware "masjidku_backend/internals/middlewares/auth"
	masjidkuMiddleware "masjidku_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func MasjidMoreAdminRoutes(router fiber.Router, db *gorm.DB) {
	// Group besar: wajib login + role admin/dkm/owner
	adminOrOwner := router.Group("/",
		authMiddleware.AuthMiddleware(db),
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola data tambahan masjid"),
			constants.AdminAndAbove, // admin, dkm, owner
		),
		masjidkuMiddleware.IsMasjidAdmin(), // inject masjid_id scope
	)

	// =========================
	// 👤 Profil DKM & Teacher (CUD only)
	// =========================
	ctrlProfile := controller.NewMasjidProfileTeacherDkmController(db)
	profilTeacherDkm := adminOrOwner.Group("/masjid-profile-teacher-dkm")
	profilTeacherDkm.Post("/", ctrlProfile.CreateProfile)
	profilTeacherDkm.Put("/:id", ctrlProfile.UpdateProfile)
	profilTeacherDkm.Delete("/:id", ctrlProfile.DeleteProfile)

	// =========================
	// 🏷️ Tag (CUD only)
	// =========================
	ctrlTag := controller.NewMasjidTagController(db)
	tag := adminOrOwner.Group("/masjid-tags")
	tag.Post("/", ctrlTag.CreateTag)
	tag.Delete("/:id", ctrlTag.DeleteTag)

	// =========================
	// 🔗 Tag Relation (CUD only)
	// =========================
	ctrlTagRelation := controller.NewMasjidTagRelationController(db)
	tagRelation := adminOrOwner.Group("/masjid-tag-relations")
	tagRelation.Post("/", ctrlTagRelation.CreateRelation)

	// =========================
	// 📊 Stats (Upsert only)
	// =========================
	ctrlStats := controller.NewMasjidStatsController(db)
	stats := adminOrOwner.Group("/masjid-stats")
	stats.Post("/", ctrlStats.UpsertStats)
}
