package routes

import (
	progressController "masjidku_backend/internals/features/progress/progress/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func UserProgressRoutes(router fiber.Router, db *gorm.DB) {
	controller := progressController.NewUserProgressController(db)
	userProgressRoutes := router.Group("/user-progress")

	userProgressRoutes.Get("/", controller.GetByUserID)
}
