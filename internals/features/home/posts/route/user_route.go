package route

import (
	"masjidku_backend/internals/features/home/posts/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AllPostRoutes(api fiber.Router, db *gorm.DB) {
	ctrl := controller.NewPostController(db)
	ctrl2 := controller.NewPostLikeController(db)
	ctrl3 := controller.NewPostThemeController(db)

	user := api.Group("/posts")
	user.Get("/", ctrl.GetAllPosts)                // 📄 Semua post publik
	user.Get("/:id", ctrl.GetPostByID)             // 🔍 Detail post
	user.Post("/by-masjid", ctrl.GetPostsByMasjid) // 🕌 Post berdasarkan masjid_id
	user.Get("/by-masjid/:slug", ctrl.GetPostsByMasjidSlug)

	post := api.Group("/post-likes")
	// 🔄 Toggle like (user harus login → ambil user_id dari token)
	post.Post("/toggle", ctrl2.ToggleLike)



theme := api.Group("/post-themes")
	// 📄 GET
	theme.Get("/", ctrl3.GetAllThemes)                  // 📄 Semua tema
	theme.Get("/:id", ctrl3.GetThemeByID)               // 🔍 Detail tema
	theme.Post("/by-masjid", ctrl3.GetThemesByMasjid)   // 🕌 Tema berdasarkan masjid
}
