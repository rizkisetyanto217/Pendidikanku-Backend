package routes

import (
	"masjidku_backend/internals/constants"
	userController "masjidku_backend/internals/features/users/user/controller"
	authMiddleware "masjidku_backend/internals/middlewares/auth"

	"github.com/go-playground/validator/v10" // ⬅️ TAMBAH
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func UserAdminRoutes(app fiber.Router, db *gorm.DB) {
	adminCtrl := userController.NewAdminUserController(db)
	userProfileCtrl := userController.NewUsersProfileController(db)
	formalCtrl := userController.NewUsersProfileFormalController(db)

	// ✅ INJECT: controller UsersTeacher (pakai validator lokal)
	v := validator.New()
	usersTeacherCtrl := userController.NewUsersTeacherController(db, v)

	// 🔐 /users – hanya teacher & above
	users := app.Group("/users",
		authMiddleware.OnlyRolesSlice(constants.RoleErrorTeacher("User Management"), constants.TeacherAndAbove),
	)

	// List & search & get by ID
	users.Get("/", adminCtrl.GetUsers)
	users.Get("/search", adminCtrl.SearchUsers)
	users.Get("/:id", adminCtrl.GetUserByID)

	// Create (single/batch) & delete (soft)
	users.Post("/", adminCtrl.CreateUser)
	users.Delete("/:id", adminCtrl.DeleteUser)

	// Admin-only: lihat deleted, restore, force delete
	users.Get("/deleted", adminCtrl.GetDeletedUsers)
	users.Post("/:id/restore", adminCtrl.RestoreUser)
	users.Delete("/:id/force", adminCtrl.ForceDeleteUser)

	// 🔐 Tambahan: admin bisa lihat semua user profile
	app.Get("/users-profiles",
		authMiddleware.OnlyRolesSlice(constants.RoleErrorTeacher("Lihat Semua User Profile"), constants.TeacherAndAbove),
		userProfileCtrl.GetProfiles,
	)

	// ✅ Admin akses formal profile by user_id
	usersFormal := app.Group("/users-profiles-formal",
		authMiddleware.OnlyRolesSlice(constants.RoleErrorTeacher("Akses Formal Profile"), constants.TeacherAndAbove),
	)
	usersFormal.Get("/:user_id", formalCtrl.AdminGetByUserID)
	usersFormal.Delete("/:user_id", formalCtrl.AdminDeleteByUserID)

	// ✅ ✅ INJECT: ADMIN ROUTES untuk users_teacher (CRUD lengkap)
	usersTeacher := app.Group("/users-teacher",
		authMiddleware.OnlyRolesSlice(constants.RoleErrorTeacher("Kelola Profil Pengajar"), constants.TeacherAndAbove),
	)
	usersTeacher.Post("/", usersTeacherCtrl.Create)
	usersTeacher.Get("/", usersTeacherCtrl.List)
	usersTeacher.Get("/:id", usersTeacherCtrl.GetByID)
	usersTeacher.Get("/by-user/:user_id", usersTeacherCtrl.GetByUserID)
	usersTeacher.Patch("/:id", usersTeacherCtrl.Update)
	usersTeacher.Delete("/:id", usersTeacherCtrl.Delete)
}
