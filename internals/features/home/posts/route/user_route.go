package route

import (
	postController "schoolku_backend/internals/features/home/posts/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// 👤 User (login wajib) – aksi like
func PostUserRoutes(router fiber.Router, db *gorm.DB) {
	likeCtrl := postController.NewPostLikeController(db)

	user := router.Group("/post-likes")

	user.Post("/:slug/toggle", likeCtrl.ToggleLike) // 🔄 Toggle like (pakai school slug)
}
