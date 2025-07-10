package route

import (
	"masjidku_backend/internals/features/home/advices/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AdviceAdminRoutes(api fiber.Router, db *gorm.DB) {
	adviceCtrl := controller.NewAdviceController(db)

	// === ADMIN ROUTES ===
	admin := api.Group("/advices")
	admin.Get("/", adviceCtrl.GetAllAdvices)                              // 📄 Admin lihat semua
	admin.Get("/by-lecture/:lectureId", adviceCtrl.GetAdvicesByLectureID) // 🔍 Filter berdasarkan kajian
	admin.Get("/by-user/:userId", adviceCtrl.GetAdvicesByUserID)          // 🔍 Filter berdasarkan user
	admin.Delete("/:id", adviceCtrl.DeleteAdvice)                         // 🗑️ Hapus saran
}
