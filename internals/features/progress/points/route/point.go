package routes

import (
	pointController "masjidku_backend/internals/features/progress/points/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func UserPointRoutes(router fiber.Router, db *gorm.DB) {
	userPointLogController := pointController.NewUserPointLogController(db)
	userPointRoutes := router.Group("/user-point-logs")

	userPointRoutes.Post("/", userPointLogController.Create)
	userPointRoutes.Get("/", userPointLogController.GetByUserID)
}
