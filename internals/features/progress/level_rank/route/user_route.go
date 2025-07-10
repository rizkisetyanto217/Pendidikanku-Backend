package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	levelController "masjidku_backend/internals/features/progress/level_rank/controller"
)

func LevelRequirementUserRoute(router fiber.Router, db *gorm.DB) {
	levelCtrl := levelController.NewLevelRequirementController(db)
	rankCtrl := levelController.NewRankRequirementController(db)

	// 🎯 Level Routes (User only - readonly)
	levelRoutes := router.Group("/level-requirements")
	levelRoutes.Get("/", levelCtrl.GetAll)
	levelRoutes.Get("/:id", levelCtrl.GetByID)

	// 🏆 Rank Routes (User only - readonly)
	rankRoutes := router.Group("/rank-requirements")
	rankRoutes.Get("/", rankCtrl.GetAll)
	rankRoutes.Get("/:id", rankCtrl.GetByID)
}
