package route

import (
	"masjidku_backend/internals/features/masjids/user_follow_masjids/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func UserFollowMasjidsRoutes(user fiber.Router, db *gorm.DB) {
	ctrl := controller.NewUserFollowMasjidController(db)

	// 🤝 Group: /user-follow-masjids
	follow := user.Group("/user-follow-masjids")
	follow.Post("/follow", ctrl.FollowMasjid)              // ➕ Follow masjid
	follow.Delete("/unfollow", ctrl.UnfollowMasjid)        // ❌ Unfollow masjid
	follow.Get("/followed", ctrl.GetFollowedMasjidsByUser) // 📄 Lihat daftar masjid yang di-follow

	// ✅ Tambahkan route baru untuk is-following
	follow.Get("/is-following", ctrl.IsFollowing)
}
