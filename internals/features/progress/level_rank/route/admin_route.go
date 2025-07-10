package route

import (
	"masjidku_backend/internals/constants"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	levelController "masjidku_backend/internals/features/progress/level_rank/controller"
	authMiddleware "masjidku_backend/internals/middlewares/auth"
)

func LevelRequirementAdminRoute(router fiber.Router, db *gorm.DB) {
	levelCtrl := levelController.NewLevelRequirementController(db)
	rankCtrl := levelController.NewRankRequirementController(db)

	// 🎯 Level Routes (Admin/Teacher only)
	levelRoutes := router.Group("/level-requirements")
	levelRoutes.Post("/", authMiddleware.OnlyRolesSlice(
		constants.RoleErrorTeacher("menambahkan level"),
		constants.TeacherAndAbove,
	), levelCtrl.Create)

	levelRoutes.Put("/:id", authMiddleware.OnlyRolesSlice(
		constants.RoleErrorTeacher("mengedit level"),
		constants.TeacherAndAbove,
	), levelCtrl.Update)

	levelRoutes.Delete("/:id", authMiddleware.OnlyRolesSlice(
		constants.RoleErrorTeacher("menghapus level"),
		constants.TeacherAndAbove,
	), levelCtrl.Delete)

	// 🏆 Rank Routes (Admin/Teacher only)
	rankRoutes := router.Group("/rank-requirements")
	rankRoutes.Post("/", authMiddleware.OnlyRolesSlice(
		constants.RoleErrorTeacher("menambahkan rank"),
		constants.TeacherAndAbove,
	), rankCtrl.Create)

	rankRoutes.Put("/:id", authMiddleware.OnlyRolesSlice(
		constants.RoleErrorTeacher("mengedit rank"),
		constants.TeacherAndAbove,
	), rankCtrl.Update)

	rankRoutes.Delete("/:id", authMiddleware.OnlyRolesSlice(
		constants.RoleErrorTeacher("menghapus rank"),
		constants.TeacherAndAbove,
	), rankCtrl.Delete)
}
