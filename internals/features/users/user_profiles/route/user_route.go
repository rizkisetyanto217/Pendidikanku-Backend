package routes

import (
	"log"

	userController "masjidku_backend/internals/features/users/user_profiles/controller"

	// ⬅️ TAMBAH
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func UserProfileUserRoutes(app fiber.Router, db *gorm.DB) {
	log.Println("[DEBUG] ❗ Masuk UserAllRoutes")

	formalCtrl := userController.NewUsersProfileFormalController(db)
	docCtrl := userController.NewUsersProfileDocumentController(db)

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

}
