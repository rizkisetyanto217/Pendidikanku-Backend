package routes

import (
	"log"

	// docController "masjidku_backend/internals/features/users/user/controller" // ⬅️ controller dokumen
	// formalController "masjidku_backend/internals/features/users/user/controller"
	userController "masjidku_backend/internals/features/users/user/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func UserAllRoutes(app fiber.Router, db *gorm.DB) {
	log.Println("[DEBUG] ❗ Masuk UserAllRoutes")

	selfCtrl := userController.NewUserSelfController(db)
	userProfileCtrl := userController.NewUsersProfileController(db)
	formalCtrl := userController.NewUsersProfileFormalController(db)
	docCtrl := userController.NewUsersProfileDocumentController(db) // ⬅️ inisialisasi

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
	docs.Post("/upload/many", docCtrl.CreateMultipartMany)    // upload banyak
	docs.Get("/", docCtrl.List)                              // list dokumen milik user
	docs.Get("/:doc_type", docCtrl.GetByDocType)             // ambil per jenis
	docs.Patch("/:doc_type/upload", docCtrl.UpdateMultipart) // update + upload baru
	docs.Delete("/:doc_type", docCtrl.DeleteSoft)            // hapus (soft/hard via query)
}
