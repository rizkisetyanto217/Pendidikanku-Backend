package routes

import (
	"masjidku_backend/internals/constants"
	userController "masjidku_backend/internals/features/users/users/controller"
	authMiddleware "masjidku_backend/internals/middlewares/auth"

	// ⬅️ TAMBAH
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func UserAdminRoutes(app fiber.Router, db *gorm.DB) {
	adminCtrl := userController.NewAdminUserController(db)
	userProfileCtrl := userController.NewUsersProfileController(db, nil)

	// 🔐 /users – hanya teacher & above
	users := app.Group("/users",
		authMiddleware.OnlyRolesSlice(constants.RoleErrorTeacher("User Management"), constants.TeacherAndAbove),
	)

	// List & search & get by ID
	users.Get("/list", adminCtrl.ListUsers)

	// Create (single/batch) & delete (soft)
	users.Post("/", adminCtrl.CreateUser)
	users.Delete("/:id", adminCtrl.DeleteUser)

	// Admin-only: lihat deleted, restore, force delete
	users.Post("/:id/restore", adminCtrl.RestoreUser)

	// 🔐 Tambahan: admin bisa lihat semua user profile
	app.Get("/users-profiles",
		authMiddleware.OnlyRolesSlice(constants.RoleErrorTeacher("Lihat Semua User Profile"), constants.TeacherAndAbove),
		userProfileCtrl.GetProfiles,
	)


}
