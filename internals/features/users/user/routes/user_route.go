package routes

import (
	"log"

	userController "masjidku_backend/internals/features/users/user/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func UserAllRoutes(app fiber.Router, db *gorm.DB) {
	log.Println("[DEBUG] ❗ Masuk UserAllRoutes")

	selfCtrl := userController.NewUserSelfController(db)
	userProfileCtrl := userController.NewUsersProfileController(db)

	// ✅ Profil diri (JWT)
	// Rekomendasi path baru:
	app.Get("/users/me", selfCtrl.GetMe)
	app.Put("/users/me", selfCtrl.UpdateMe)

	// (Opsional) Backward-compat: route lama tetap diarahkan ke handler baru
	// app.Get("/users/user", selfCtrl.GetMe)

	// ✅ Profile data (punyamu tetap)
	app.Get("/users-profiles/me", userProfileCtrl.GetProfile)
	app.Post("/users-profiles/save", userProfileCtrl.CreateProfile)
	app.Put("/users-profiles", userProfileCtrl.UpdateProfile)
	app.Delete("/users-profiles", userProfileCtrl.DeleteProfile)
}
