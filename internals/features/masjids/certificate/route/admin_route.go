package route

import (
	certificateController "masjidku_backend/internals/features/masjids/certificate/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	masjidkuMiddleware "masjidku_backend/internals/middlewares/features"
)

func CertificateAdminRoutes(dkm fiber.Router, db *gorm.DB) {
	certificateCtrl := certificateController.NewCertificateController(db)

	dkm.Post("/certificates",
		masjidkuMiddleware.IsMasjidAdmin(),
		certificateCtrl.Create,
	)

	dkm.Get("/certificates", certificateCtrl.GetAll)
	dkm.Get("/certificates/:id", certificateCtrl.GetByID)

	dkm.Put("/certificates/:id",
		masjidkuMiddleware.IsMasjidAdmin(),
		certificateCtrl.Update,
	)

	dkm.Delete("/certificates/:id",
		masjidkuMiddleware.IsMasjidAdmin(),
		certificateCtrl.Delete,
	)
}
