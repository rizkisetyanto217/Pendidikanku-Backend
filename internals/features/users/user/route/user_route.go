package routes

import (
	"log"

	userController "masjidku_backend/internals/features/users/user/controller"

	// ⬅️ TAMBAH
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func UserUserRoutes(app fiber.Router, db *gorm.DB) {
	log.Println("[DEBUG] ❗ Masuk UserAllRoutes")

	selfCtrl := userController.NewUserSelfController(db)
	userProfileCtrl := userController.NewUsersProfileController(db)


	// ✅ Profil diri (JWT)
	app.Get("/users/me", selfCtrl.GetMe)
	app.Patch("/users/me", selfCtrl.UpdateMe)

	// ✅ Profile data (existing)
	app.Get("/users-profiles/me", userProfileCtrl.GetProfile)
	app.Post("/users-profiles/save", userProfileCtrl.CreateProfile)
	app.Patch("/users-profiles", userProfileCtrl.UpdateProfile)
	app.Delete("/users-profiles", userProfileCtrl.DeleteProfile)


}
