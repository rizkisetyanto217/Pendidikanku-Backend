package routes

import (
	"masjidku_backend/internals/features/users/token/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func RegisterTokenRoutes(router fiber.Router, db *gorm.DB) {
	tokenController := controller.NewTokenController(db)

	routes := router.Group("/tokens") // hasil akhir: /api/u/tokens/...
	routes.Get("/", tokenController.GetAll)
	routes.Get("/:id", tokenController.GetByID)
	routes.Post("/", tokenController.Create)
	routes.Put("/:id", tokenController.Update)
	routes.Delete("/:id", tokenController.Delete)
}
