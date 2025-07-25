package route

import (
	certificateController "masjidku_backend/internals/features/masjids/certificate/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)


func PublicCertificateRoutes(router fiber.Router, db *gorm.DB) {
	certCtrl := certificateController.NewUserCertificateController(db)

	certificateCtrl := certificateController.NewCertificateController(db)

	// Grouping: /certificates
	cert := router.Group("/certificates")
	// GET - publik
	cert.Get("/", certificateCtrl.GetAll)
	cert.Get("/:id", certificateCtrl.GetByID)
	cert.Get("/by-user-exam/:user_exam_id", certificateCtrl.GetByUserExamID)


	certs := router.Group("/user-certificates")
	// 🌐 Public / All User
	certs.Get("/", certCtrl.GetAll)                   // 📄 Lihat semua sertifikat
	certs.Get("/:id", certCtrl.GetByID)               // 🔍 Detail sertifikat by ID
	certs.Post("/", certCtrl.Create)                  // ➕ Buat sertifikat
	// ✅ User sendiri (harus login)
	certs.Put("/:id", certCtrl.Update) // ✏️ Update hasil/slug sertifikat

}
