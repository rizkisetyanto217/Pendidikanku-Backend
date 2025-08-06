package route

import (
	"masjidku_backend/internals/features/masjids/lecture_sessions/materials/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// 👥 User Routes (Read-only)
func AllLectureSessionsAssetRoutes(user fiber.Router, db *gorm.DB) {
	assetCtrl := controller.NewLectureSessionsAssetController(db)
	materialCtrl := controller.NewLectureSessionsMaterialController(db)

	// 📁 Group: /lecture-sessions-assets
	asset := user.Group("/lecture-sessions-assets")
	asset.Get("/", assetCtrl.GetAllLectureSessionsAssets)    // 📄 Lihat semua asset
	asset.Get("/filter", assetCtrl.FilterLectureLectureSessionsAssets)
	asset.Get("filter-slug", assetCtrl.FilterLectureSessionsAssetsBySlug)
	asset.Get("/filter-by-lecture-id", assetCtrl.FindGroupedByLectureID)
	asset.Get("/filter-by-lecture-slug", assetCtrl.FindGroupedByLectureSlug)

	// 📚 Group: /lecture-sessions-materials
	material := user.Group("/lecture-sessions-materials")
	material.Get("/", materialCtrl.GetAllLectureSessionsMaterials)    // 📄 Semua materi
	material.Get("/filter", materialCtrl.FindByLectureSessionFiltered)
	material.Get("/filter-slug", materialCtrl.FindByLectureSessionFilteredBySlug)
	material.Get("/get-by-id/:id", materialCtrl.GetLectureSessionsMaterialByID) // 🔍 Detail materi
	material.Get("/filter-by-lecture-id", materialCtrl.FindGroupedMaterialsByLectureID)
	material.Get("/filter-by-lecture-slug", materialCtrl.FindGroupedMaterialsByLectureSlug)


}