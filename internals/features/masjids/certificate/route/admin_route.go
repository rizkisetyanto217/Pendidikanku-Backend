package route

import (
	certificateController "masjidku_backend/internals/features/masjids/certificate/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	masjidkuMiddleware "masjidku_backend/internals/middlewares/features"
)

func CertificateAdminRoutes(router fiber.Router, db *gorm.DB) {
	certificateCtrl := certificateController.NewCertificateController(db)

	// Grouping: /certificates
	cert := router.Group("/certificates")

	// GET - publik
	cert.Get("/", certificateCtrl.GetAll)
	cert.Get("/:id", certificateCtrl.GetByID)

	// Admin only
	cert.Post("/", masjidkuMiddleware.IsMasjidAdmin(), certificateCtrl.Create)
	cert.Put("/:id", masjidkuMiddleware.IsMasjidAdmin(), certificateCtrl.Update)
	cert.Delete("/:id", masjidkuMiddleware.IsMasjidAdmin(), certificateCtrl.Delete)
}
