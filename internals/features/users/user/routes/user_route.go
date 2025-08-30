package routes

import (
	"log"

	userController "masjidku_backend/internals/features/users/user/controller"

	"github.com/go-playground/validator/v10" // ⬅️ TAMBAH
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func UserUserRoutes(app fiber.Router, db *gorm.DB) {
	log.Println("[DEBUG] ❗ Masuk UserAllRoutes")

	selfCtrl := userController.NewUserSelfController(db)
	userProfileCtrl := userController.NewUsersProfileController(db)
	formalCtrl := userController.NewUsersProfileFormalController(db)
	docCtrl := userController.NewUsersProfileDocumentController(db)

	// ✅ INJECT: controller UsersTeacher (pakai validator lokal)
	v := validator.New()
	usersTeacherCtrl := userController.NewUsersTeacherController(db, v)

	// ✅ Profil diri (JWT)
	app.Get("/users/me", selfCtrl.GetMe)
	app.Patch("/users/me", selfCtrl.UpdateMe)

	// ✅ Profile data (existing)
	app.Get("/users-profiles/me", userProfileCtrl.GetProfile)
	app.Post("/users-profiles/save", userProfileCtrl.CreateProfile)
	app.Patch("/users-profiles", userProfileCtrl.UpdateProfile)
	app.Delete("/users-profiles", userProfileCtrl.DeleteProfile)

	// ✅ Formal profile (punya sendiri)
	app.Get("/users-profiles-formal", formalCtrl.GetMine)
	app.Put("/users-profiles-formal", formalCtrl.UpsertMine)
	app.Patch("/users-profiles-formal", formalCtrl.PatchMine)
	app.Delete("/users-profiles-formal", formalCtrl.DeleteMine)

	// ✅ Documents (punya sendiri)
	docs := app.Group("/users/profile/documents")
	docs.Post("/upload/many", docCtrl.CreateMultipartMany)
	docs.Get("/", docCtrl.List)
	docs.Get("/:doc_type", docCtrl.GetByDocType)
	docs.Patch("/:doc_type/upload", docCtrl.UpdateMultipart)
	docs.Delete("/:doc_type", docCtrl.DeleteSoft)

	// ✅ ✅ INJECT: PUBLIC/GENERAL READ untuk users_teacher
	app.Get("/users-teacher", usersTeacherCtrl.List)
	app.Get("/users-teacher/:id", usersTeacherCtrl.GetByID)
	app.Get("/users-teacher/by-user/:user_id", usersTeacherCtrl.GetByUserID)
}
