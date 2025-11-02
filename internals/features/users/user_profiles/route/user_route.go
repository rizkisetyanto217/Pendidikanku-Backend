package routes

import (
	"log"

	userController "schoolku_backend/internals/features/users/user_profiles/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func UserProfileUserRoutes(r fiber.Router, db *gorm.DB) {
	log.Println("[DEBUG] ❗ Masuk UserProfileUserRoutes")

	formalCtrl := userController.NewUsersProfileFormalController(db)
	docCtrl := userController.NewUsersProfileDocumentController(db)

	// Base: /users/profile
	base := r.Group("/users/profile")

	// /users/profile/formal  (punya sendiri)
	formal := base.Group("/formal")
	formal.Get("/", formalCtrl.GetMine)
	formal.Put("/", formalCtrl.UpsertMine)
	formal.Patch("/", formalCtrl.PatchMine)
	formal.Delete("/", formalCtrl.DeleteMine)

	// /users/profile/documents (punya sendiri)
	docs := base.Group("/documents")
	docs.Get("", docCtrl.List)                               // GET /users/profile/documents
	docs.Get("/:doc_type", docCtrl.GetByDocType)             // GET /users/profile/documents/:doc_type
	docs.Post("/upload/many", docCtrl.CreateMultipartMany)   // POST /users/profile/documents/upload/many
	docs.Patch("/:doc_type/upload", docCtrl.UpdateMultipart) // PATCH /users/profile/documents/:doc_type/upload
	docs.Delete("/:doc_type", docCtrl.DeleteSoft)            // DELETE /users/profile/documents/:doc_type
}
