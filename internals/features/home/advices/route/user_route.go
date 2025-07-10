package route

import (
	"masjidku_backend/internals/features/home/advices/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AllAdviceRoutes(api fiber.Router, db *gorm.DB) {
	adviceCtrl := controller.NewAdviceController(db)

	user := api.Group("/advices")
	user.Post("/", adviceCtrl.CreateAdvice)
	user.Get("/by-lecture/:lectureId", adviceCtrl.GetAdvicesByLectureID)
	user.Get("/by-user/:userId", adviceCtrl.GetAdvicesByUserID)
}
