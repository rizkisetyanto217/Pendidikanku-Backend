package route

import (
	certcontroller "masjidku_backend/internals/features/masjids/certificate/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)


func PublicCertificateRoutes(user fiber.Router, db *gorm.DB) {
	certCtrl := certcontroller.NewUserCertificateController(db)

	certs := user.Group("/user-certificates")

	// 🌐 Public / All User
	certs.Get("/", certCtrl.GetAll)                   // 📄 Lihat semua sertifikat
	certs.Get("/:id", certCtrl.GetByID)               // 🔍 Detail sertifikat by ID
	certs.Post("/", certCtrl.Create)                  // ➕ Buat sertifikat

	// ✅ User sendiri (harus login)
	certs.Put("/:id", certCtrl.Update) // ✏️ Update hasil/slug sertifikat

}
