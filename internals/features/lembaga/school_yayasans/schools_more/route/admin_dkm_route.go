package route

import (
	"madinahsalam_backend/internals/constants"
	"madinahsalam_backend/internals/features/lembaga/school_yayasans/schools_more/controller"
	authMiddleware "madinahsalam_backend/internals/middlewares/auth"
	schoolkuMiddleware "madinahsalam_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func SchoolMoreAdminRoutes(router fiber.Router, db *gorm.DB) {
	// Group besar: wajib login + role admin/dkm/owner
	adminOrOwner := router.Group("/",
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola data tambahan school"),
			constants.AdminAndAbove, // admin, dkm, owner
		),
		schoolkuMiddleware.IsSchoolAdmin(), // inject school_id scope
	)

	// =========================
	// 👤 Profil DKM & Teacher (CUD only)
	// =========================
	ctrlProfile := controller.NewSchoolProfileTeacherDkmController(db)
	profilTeacherDkm := adminOrOwner.Group("/school-profile-teacher-dkm")
	profilTeacherDkm.Post("/", ctrlProfile.CreateProfile)
	profilTeacherDkm.Put("/:id", ctrlProfile.UpdateProfile)
	profilTeacherDkm.Delete("/:id", ctrlProfile.DeleteProfile)

	// =========================
	// 🏷️ Tag (CUD only)
	// =========================
	ctrlTag := controller.NewSchoolTagController(db)
	tag := adminOrOwner.Group("/school-tags")
	tag.Post("/", ctrlTag.CreateTag)
	tag.Delete("/:id", ctrlTag.DeleteTag)

	// =========================
	// 🔗 Tag Relation (CUD only)
	// =========================
	ctrlTagRelation := controller.NewSchoolTagRelationController(db)
	tagRelation := adminOrOwner.Group("/school-tag-relations")
	tagRelation.Post("/", ctrlTagRelation.CreateRelation)

	// =========================
	// 📊 Stats (Upsert only)
	// =========================
	ctrlStats := controller.NewSchoolStatsController(db)
	stats := adminOrOwner.Group("/school-stats")
	stats.Post("/", ctrlStats.UpsertStats)
}
