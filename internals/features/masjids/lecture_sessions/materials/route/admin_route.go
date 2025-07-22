package route

import (
	"masjidku_backend/internals/features/masjids/lecture_sessions/materials/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// 🔐 Admin Routes (CRUD)
func LectureSessionsAssetAdminRoutes(admin fiber.Router, db *gorm.DB) {
	assetCtrl := controller.NewLectureSessionsAssetController(db)
	materialCtrl := controller.NewLectureSessionsMaterialController(db)

	// 📁 Group: /lecture-sessions-assets
	asset := admin.Group("/lecture-sessions-assets")
	asset.Post("/", assetCtrl.CreateLectureSessionsAsset)      // ➕ Tambah asset
	asset.Get("/", assetCtrl.GetAllLectureSessionsAssets)      // 📄 Lihat semua asset
	asset.Get("/filter", assetCtrl.FilterLectureSessionsAssets)
	asset.Get("/:id", assetCtrl.GetLectureSessionsAssetByID)

	asset.Delete("/:id", assetCtrl.DeleteLectureSessionsAsset) // ❌ Hapus asset

	// 📚 Group: /lecture-sessions-materials
	material := admin.Group("/lecture-sessions-materials")

	material.Post("/", materialCtrl.CreateLectureSessionsMaterial)        // ➕ Tambah materi
	material.Get("/", materialCtrl.GetAllLectureSessionsMaterials)        // 📄 Semua materi
	material.Get("/filter", materialCtrl.FindByLectureSessionFiltered)    // ✅ Filter (tambahkan kalau perlu)
	material.Get("/get-by-id/:id", materialCtrl.GetLectureSessionsMaterialByID) // ✅ Lebih aman
	material.Delete("/:id", materialCtrl.DeleteLectureSessionsMaterial)   // ❌ Hapus materi

}
