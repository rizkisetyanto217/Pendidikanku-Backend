package route

import (
	certificateController "schoolku_backend/internals/features/schools/certificate/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Publik: baca master certificate (read-only)
func AllCertificateRoutes(api fiber.Router, db *gorm.DB) {
	certCtrl := certificateController.NewCertificateController(db)
	certs := api.Group("/certificates")
	certs.Get("/", certCtrl.GetAll)
	certs.Get("/:id", certCtrl.GetByID)
	certs.Get("/by-user-exam/:user_exam_id", certCtrl.GetByUserExamID) // ← jika memang aman untuk publik
}
