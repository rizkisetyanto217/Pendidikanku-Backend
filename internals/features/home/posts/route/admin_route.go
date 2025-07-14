package route

import (
	"masjidku_backend/internals/features/home/posts/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func PostAdminRoutes(api fiber.Router, db *gorm.DB) {
	ctrl := controller.NewPostController(db)
	ctrl2 := controller.NewPostThemeController(db)

	admin := api.Group("/posts")
	admin.Post("/", ctrl.CreatePost)       // ➕ Buat post (user masjid)
	admin.Put("/:id", ctrl.UpdatePost)    // ✏️ Update post (admin)
	admin.Delete("/:id", ctrl.DeletePost) // 🗑️ Hapus post
	// Admin bisa lihat semua post juga (jika butuh)
	admin.Get("/", ctrl.GetAllPosts)    // 📄 Semua post
	admin.Get("/:id", ctrl.GetPostByID) // 🔍 Detail post

	theme := api.Group("/post-themes")

	// ➕ CRUD untuk tema (admin masjid)
	theme.Post("/", ctrl2.CreateTheme)       // ➕ Buat tema
	theme.Put("/:id", ctrl2.UpdateTheme)     // ✏️ Update tema
	theme.Delete("/:id", ctrl2.DeleteTheme)  // 🗑️ Hapus tema


	
}
