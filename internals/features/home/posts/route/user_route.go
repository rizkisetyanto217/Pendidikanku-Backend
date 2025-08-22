package route

import (
	postController "masjidku_backend/internals/features/home/posts/controller"
	authMiddleware "masjidku_backend/internals/middlewares/auth"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// 👤 User (login wajib) – aksi like
func PostUserRoutes(router fiber.Router, db *gorm.DB) {
	likeCtrl := postController.NewPostLikeController(db)

	user := router.Group("/post-likes",
		authMiddleware.AuthMiddleware(db),
	)

	user.Post("/:slug/toggle", likeCtrl.ToggleLike) // 🔄 Toggle like (pakai masjid slug)
}
