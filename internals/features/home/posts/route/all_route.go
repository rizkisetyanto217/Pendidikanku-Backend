package route

import (
	postController "masjidku_backend/internals/features/home/posts/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// 🌐 Public (read-only)
func AllPublicRoutes(router fiber.Router, db *gorm.DB) {
	postCtrl := postController.NewPostController(db)
	likeCtrl := postController.NewPostLikeController(db)
	themeCtrl := postController.NewPostThemeController(db)

	// === /posts (read)
	posts := router.Group("/posts")
	posts.Get("/", postCtrl.GetAllPosts)                   // 📄 Semua post publik
	posts.Post("/by-masjid", postCtrl.GetPostsByMasjid)    // 🕌 Berdasarkan masjid_id (body)
	posts.Get("/by-masjid/:slug", postCtrl.GetPostsByMasjidSlug)
	posts.Get("/:id", postCtrl.GetPostByID)                // 🔍 Detail post

	// === /post-likes (read-only)
	likes := router.Group("/post-likes")
	likes.Get("/:post_id/all", likeCtrl.GetAllLikesByPost) // 🔍 Semua like by post_id

	// === /post-themes (read-only)
	themes := router.Group("/post-themes")
	themes.Get("/", themeCtrl.GetAllThemes)                 // 📄 Semua tema
	themes.Get("/:id", themeCtrl.GetThemeByID)              // 🔍 Detail tema
	themes.Post("/by-masjid", themeCtrl.GetThemesByMasjid)  // 🕌 Tema by masjid (body)
}
