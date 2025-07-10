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
	contentCtrl := controller.NewLectureSessionsContentController(db)

	// 📁 Group: /lecture-sessions-assets
	asset := user.Group("/lecture-sessions-assets")
	asset.Get("/", assetCtrl.GetAllLectureSessionsAssets)    // 📄 Lihat semua asset
	asset.Get("/:id", assetCtrl.GetLectureSessionsAssetByID) // 🔍 Detail asset

	// 📚 Group: /lecture-sessions-materials
	material := user.Group("/lecture-sessions-materials")
	material.Get("/", materialCtrl.GetAllLectureSessionsMaterials)    // 📄 Semua materi
	material.Get("/:id", materialCtrl.GetLectureSessionsMaterialByID) // 🔍 Detail materi

	// 🧩 Group: /lecture-sessions-content
	content := user.Group("/lecture-sessions-content")
	content.Get("/by-lecture", contentCtrl.GetContentByLectureID) // 🔀 GET /lecture-sessions-content/by-lecture?lecture_id=...
}
