package route

import (
	"masjidku_backend/internals/constants"
	materialController "masjidku_backend/internals/features/masjids/lecture_sessions/materials/controller"
	authMiddleware "masjidku_backend/internals/middlewares/auth"
	masjidkuMiddleware "masjidku_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// 🔐 Admin/DKM/Owner (CRUD)
func LectureSessionsAssetAdminRoutes(router fiber.Router, db *gorm.DB) {
	assetCtrl := materialController.NewLectureSessionsAssetController(db)
	materialCtrl := materialController.NewLectureSessionsMaterialController(db)

	// Group besar: wajib login + role admin/dkm/owner + scope masjid
	adminOrOwner := router.Group("/",
		authMiddleware.AuthMiddleware(db),
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola asset & materi sesi kajian"),
			constants.AdminAndAbove, // admin, dkm, owner
		),
		masjidkuMiddleware.IsMasjidAdmin(), // inject masjid_id scope
	)

	// 📁 /lecture-sessions-assets (CRUD)
	asset := adminOrOwner.Group("/lecture-sessions-assets")
	asset.Post("/", assetCtrl.CreateLectureSessionsAsset)        // ➕ Tambah asset
	asset.Get("/", assetCtrl.GetAllLectureSessionsAssets)        // 📄 Lihat semua asset (scoped)
	asset.Get("/:id", assetCtrl.GetLectureSessionsAssetByID)     // 🔍 Detail asset
	asset.Put("/:id", assetCtrl.UpdateLectureSessionsAsset)      // ✏️ Ubah asset
	asset.Delete("/:id", assetCtrl.DeleteLectureSessionsAsset)   // ❌ Hapus asset

	// 📚 /lecture-sessions-materials (CRUD)
	material := adminOrOwner.Group("/lecture-sessions-materials")
	material.Post("/", materialCtrl.CreateLectureSessionsMaterial)                 // ➕ Tambah materi
	material.Get("/", materialCtrl.GetAllLectureSessionsMaterials)                 // 📄 Semua materi (scoped)
	material.Get("/filter", materialCtrl.FindByLectureSessionFiltered)             // 🔎 Filter (opsional)
	material.Get("/get-by-id/:id", materialCtrl.GetLectureSessionsMaterialByID)    // 🔍 Detail materi
	material.Put("/:id", materialCtrl.UpdateLectureSessionsMaterial)               // ✏️ Ubah materi
	material.Delete("/:id", materialCtrl.DeleteLectureSessionsMaterial)            // ❌ Hapus materi
}
