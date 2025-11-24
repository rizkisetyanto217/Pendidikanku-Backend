package route

// import (
// 	"madinahsalam_backend/internals/features/lembaga/school_yayasans/user_follow_schools/controller"

// 	"github.com/gofiber/fiber/v2"
// 	"gorm.io/gorm"
// )

// func UserFollowSchoolsRoutes(user fiber.Router, db *gorm.DB) {
// 	ctrl := controller.NewUserFollowSchoolController(db)

// 	// 🤝 Group: /user-follow-schools
// 	follow := user.Group("/user-follow-schools")
// 	follow.Post("/follow", ctrl.FollowSchool)              // ➕ Follow school
// 	follow.Delete("/unfollow", ctrl.UnfollowSchool)        // ❌ Unfollow school
// 	follow.Get("/followed", ctrl.GetFollowedSchoolsByUser) // 📄 Lihat daftar school yang di-follow

// 	// ✅ Tambahkan route baru untuk is-following
// 	follow.Get("/is-following", ctrl.IsFollowing)
// }
