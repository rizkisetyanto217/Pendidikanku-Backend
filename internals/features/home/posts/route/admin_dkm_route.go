package route

import (
	"schoolku_backend/internals/constants"
	postController "schoolku_backend/internals/features/home/posts/controller"
	authMiddleware "schoolku_backend/internals/middlewares/auth"
	schoolkuMiddleware "schoolku_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// 🔐 Admin/DKM/Owner – kelola post & tema (scoped by school)
func PostAdminRoutes(router fiber.Router, db *gorm.DB) {
	postCtrl := postController.NewPostController(db)
	themeCtrl := postController.NewPostThemeController(db)

	admin := router.Group("/",
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola postingan & tema"),
			constants.AdminAndAbove, // admin, dkm, owner
		),
		schoolkuMiddleware.IsSchoolAdmin(), // inject school_id scope
	)

	// === /posts (CRUD + list scoped)
	posts := admin.Group("/posts")
	posts.Post("/", postCtrl.CreatePost)      // ➕ Buat post
	posts.Put("/:id", postCtrl.UpdatePost)    // ✏️ Update post
	posts.Delete("/:id", postCtrl.DeletePost) // 🗑️ Hapus post
	posts.Get("/", postCtrl.GetAllPosts)      // 📄 Semua post (scoped)
	posts.Get("/by-school", postCtrl.GetPostsBySchool)

	// === /post-themes (CRUD + list scoped)
	themes := admin.Group("/post-themes")
	themes.Get("/by-school", themeCtrl.GetThemesBySchool)
	themes.Post("/", themeCtrl.CreateTheme)
	themes.Put("/:id", themeCtrl.UpdateTheme)
	themes.Delete("/:id", themeCtrl.DeleteTheme)
}
