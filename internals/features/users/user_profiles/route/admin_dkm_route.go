package routes

import (
	"masjidku_backend/internals/constants"
	userController "masjidku_backend/internals/features/users/user_profiles/controller"
	authMiddleware "masjidku_backend/internals/middlewares/auth"

	// ⬅️ TAMBAH
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func UserProfileAdminRoutes(app fiber.Router, db *gorm.DB) {
	formalCtrl := userController.NewUsersProfileFormalController(db)


	// ✅ Admin akses formal profile by user_id
	usersFormal := app.Group("/users-profiles-formal",
		authMiddleware.OnlyRolesSlice(constants.RoleErrorTeacher("Akses Formal Profile"), constants.TeacherAndAbove),
	)
	usersFormal.Get("/:user_id", formalCtrl.AdminGetByUserID)
	usersFormal.Delete("/:user_id", formalCtrl.AdminDeleteByUserID)

}
