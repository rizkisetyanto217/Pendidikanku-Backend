package route

import (
	"masjidku_backend/internals/constants"
	postController "masjidku_backend/internals/features/home/posts/controller"
	authMiddleware "masjidku_backend/internals/middlewares/auth"
	masjidkuMiddleware "masjidku_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// 🔐 Admin/DKM/Owner – kelola post & tema (scoped by masjid)
func PostAdminRoutes(router fiber.Router, db *gorm.DB) {
	postCtrl := postController.NewPostController(db)
	themeCtrl := postController.NewPostThemeController(db)

	admin := router.Group("/",
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola postingan & tema"),
			constants.AdminAndAbove, // admin, dkm, owner
		),
		masjidkuMiddleware.IsMasjidAdmin(), // inject masjid_id scope
	)

	// === /posts (CRUD + list scoped)
	posts := admin.Group("/posts")
	posts.Post("/", postCtrl.CreatePost)      // ➕ Buat post
	posts.Put("/:id", postCtrl.UpdatePost)    // ✏️ Update post
	posts.Delete("/:id", postCtrl.DeletePost) // 🗑️ Hapus post
	posts.Get("/", postCtrl.GetAllPosts)      // 📄 Semua post (scoped)
	posts.Get("/by-masjid", postCtrl.GetPostsByMasjid)

	// === /post-themes (CRUD + list scoped)
	themes := admin.Group("/post-themes")
	themes.Get("/by-masjid", themeCtrl.GetThemesByMasjid)
	themes.Post("/", themeCtrl.CreateTheme)
	themes.Put("/:id", themeCtrl.UpdateTheme)
	themes.Delete("/:id", themeCtrl.DeleteTheme)
}
