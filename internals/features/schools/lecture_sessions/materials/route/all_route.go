package route

import (
	materialController "schoolku_backend/internals/features/schools/lecture_sessions/materials/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// 👥 User/Publik (Read-only)
func AllLectureSessionsAssetRoutes(router fiber.Router, db *gorm.DB) {
	assetCtrl := materialController.NewLectureSessionsAssetController(db)
	materialCtrl := materialController.NewLectureSessionsMaterialController(db)

	// 📁 /lecture-sessions-assets (read-only)
	asset := router.Group("/lecture-sessions-assets")
	asset.Get("/", assetCtrl.GetAllLectureSessionsAssets)              // 📄 Semua asset
	asset.Get("/filter", assetCtrl.FilterLectureLectureSessionsAssets) // 🔎 Filter by params
	asset.Get("/filter-slug", assetCtrl.FilterLectureSessionsAssetsBySlug)
	asset.Get("/filter-by-lecture-id", assetCtrl.FindGroupedByLectureID)
	asset.Get("/filter-by-lecture-slug", assetCtrl.FindGroupedByLectureSlug)

	// 📚 /lecture-sessions-materials (read-only)
	material := router.Group("/lecture-sessions-materials")
	material.Get("/", materialCtrl.GetAllLectureSessionsMaterials)                // 📄 Semua materi
	material.Get("/filter", materialCtrl.FindByLectureSessionFiltered)            // 🔎 Filter by session
	material.Get("/filter-slug", materialCtrl.FindByLectureSessionFilteredBySlug) // 🔎 Filter by slug
	material.Get("/get-by-id/:id", materialCtrl.GetLectureSessionsMaterialByID)   // 🔍 Detail materi
	material.Get("/filter-by-lecture-id", materialCtrl.FindGroupedMaterialsByLectureID)
	material.Get("/filter-by-lecture-slug", materialCtrl.FindGroupedMaterialsByLectureSlug)
}
