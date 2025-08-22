// file: internals/routes/lecture_user_routes.go
package route

import (
	"masjidku_backend/internals/features/masjids/lectures/main/controller"
	authMiddleware "masjidku_backend/internals/middlewares/auth"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Mount di parent group: /api/v1/u (sudah dipagari AuthMiddleware di parent)
func LectureUserRoutes(api fiber.Router, db *gorm.DB) {
	userLectureCtrl := controller.NewUserLectureController(db)
	userLecture := api.Group("/user-lectures",
		authMiddleware.AuthMiddleware(db), // kalau parent /u sudah auth, baris ini opsional
	)

	// Aksi user terautentikasi (mis. mendaftar ke lecture, lihat peserta di lecture miliknya, dsb.)
	userLecture.Post("/", userLectureCtrl.CreateUserLecture)
	userLecture.Post("/by-lecture", userLectureCtrl.GetUsersByLecture)
}
