package route

import (
	"masjidku_backend/internals/constants"
	homeController "masjidku_backend/internals/features/home/advices/controller"
	authMiddleware "masjidku_backend/internals/middlewares/auth"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// 🔐 Admin/DKM/Owner (monitor & moderasi)
func AdviceAdminRoutes(router fiber.Router, db *gorm.DB) {
	adviceCtrl := homeController.NewAdviceController(db)

	admin := router.Group("/advices",
		authMiddleware.AuthMiddleware(db),
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola saran"),
			constants.AdminAndAbove, // admin, dkm, owner
		),
	)

	admin.Get("/", adviceCtrl.GetAllAdvices)                                // 📄 Semua saran
	admin.Get("/by-lecture/:lectureId", adviceCtrl.GetAdvicesByLectureID)   // 🔎 Filter per kajian
	admin.Get("/by-user/:userId", adviceCtrl.GetAdvicesByUserID)            // 🔎 Filter per user
	admin.Delete("/:id", adviceCtrl.DeleteAdvice)                           // 🗑️ Hapus saran
}
