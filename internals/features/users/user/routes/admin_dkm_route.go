package routes

import (
	"masjidku_backend/internals/constants" // ⬅️ ini tambahin
	// formalController "masjidku_backend/internals/features/users/user/controller"
	userController "masjidku_backend/internals/features/users/user/controller"
	authMiddleware "masjidku_backend/internals/middlewares/auth"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func UserAdminRoutes(app fiber.Router, db *gorm.DB) {
	adminCtrl := userController.NewAdminUserController(db)
	userProfileCtrl := userController.NewUsersProfileController(db)
	formalCtrl := userController.NewUsersProfileFormalController(db)
	// docCtrl := docController.NewUsersProfileDocumentController(db) // ⬅️ inisialisasi

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

	// // ✅ NEW: Admin akses dokumen profile by user_id
	// usersDocs := app.Group("/users-profile-documents",
	// 	authMiddleware.OnlyRolesSlice(constants.RoleErrorTeacher("Akses Dokumen User"), constants.TeacherAndAbove),
	// )
	// // list semua dokumen milik user tertentu
	// usersDocs.Get("/:user_id", docCtrl.AdminListByUserID)
	// // hapus dokumen tertentu milik user_id
	// usersDocs.Delete("/:user_id/:doc_type", docCtrl.AdminDeleteByUserIDAndType)
}
