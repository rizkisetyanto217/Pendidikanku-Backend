package routes

import (
	"log"

	userController "masjidku_backend/internals/features/users/users/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func UserUserRoutes(app fiber.Router, db *gorm.DB) {
	log.Println("[DEBUG] ❗ Masuk UserUserRoutes")

	selfCtrl := userController.NewUserSelfController(db)
	userProfileCtrl := userController.NewUsersProfileController(db, nil)

	// ===== /users/me (JWT; data diri) =====
	app.Get("/users/me", selfCtrl.GetMe)
	app.Patch("/users/me", selfCtrl.UpdateMe)

	// ===== /users/profile (JWT; profile milik sendiri) =====
	profile := app.Group("/user-profile")
	profile.Get("/",    userProfileCtrl.GetProfile)     // GET   /users/profile
	profile.Get("/:user_id", userProfileCtrl.GetProfile) // GET   /users/profile/:id
	profile.Post("/",   userProfileCtrl.CreateProfile)  // POST  /users/profile
	profile.Patch("/",  userProfileCtrl.UpdateProfile)  // PATCH /users/profile
	profile.Delete("/", userProfileCtrl.DeleteProfile)  // DELETE /users/profile
}
