package route

import (
	homeController "masjidku_backend/internals/features/home/advices/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// 👤 User (buat saran & akses miliknya)
func AllAdviceRoutes(router fiber.Router, db *gorm.DB) {
	adviceCtrl := homeController.NewAdviceController(db)

	user := router.Group("/advices",
	)

	user.Post("/", adviceCtrl.CreateAdvice)                 // ➕ Buat saran (oleh user)
	user.Get("/by-lecture/:lectureId", adviceCtrl.GetAdvicesByLectureID) // 🔎 Lihat saran per kajian
	user.Get("/by-user/:userId", adviceCtrl.GetAdvicesByUserID)          // 🔎 Lihat saran milik user
}
