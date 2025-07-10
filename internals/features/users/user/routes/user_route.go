package routes

import (
	"log"
	userController "masjidku_backend/internals/features/users/user/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func UserAllRoutes(app fiber.Router, db *gorm.DB) {
	log.Println("[DEBUG] ❗ Masuk UserAllRoutes")

	userCtrl := userController.NewUserController(db)
	userProfileCtrl := userController.NewUsersProfileController(db)

	// ✅ Bisa diakses semua user login
	app.Get("/users/user", userCtrl.GetUser)

	// ✅ Untuk semua user login (profile data)
	app.Get("/users-profiles/me", userProfileCtrl.GetProfile)
	app.Post("/users-profiles/save", userProfileCtrl.CreateProfile)
	app.Put("/users-profiles", userProfileCtrl.UpdateProfile)
	app.Delete("/users-profiles", userProfileCtrl.DeleteProfile)

}
