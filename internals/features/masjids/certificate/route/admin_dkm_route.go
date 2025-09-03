package route

import (
	"masjidku_backend/internals/constants"
	certificateController "masjidku_backend/internals/features/masjids/certificate/controller"
	authMiddleware "masjidku_backend/internals/middlewares/auth"
	masjidkuMiddleware "masjidku_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Admin/DKM/Owner: kelola master certificate + user certificates (internal)
func CertificateAdminRoutes(api fiber.Router, db *gorm.DB) {
	admin := api.Group("/",
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola sertifikat"),
			constants.AdminAndAbove, // admin, dkm, owner
		),
		masjidkuMiddleware.IsMasjidAdmin(), // inject/cek masjid_id dari token
	)

	// Master Certificate (template/definisi)
	certCtrl := certificateController.NewCertificateController(db)
	certs := admin.Group("/certificates")
	certs.Get("/", certCtrl.GetAll)
	certs.Get("/:id", certCtrl.GetByID)
	certs.Post("/", certCtrl.Create)
	certs.Put("/:id", certCtrl.Update)
	certs.Delete("/:id", certCtrl.Delete)

	// User Certificates (penerbitan/rekap internal)
	userCertCtrl := certificateController.NewUserCertificateController(db)
	uc := admin.Group("/user-certificates")
	uc.Get("/", userCertCtrl.GetAll)   // daftar internal (scoped)
	uc.Get("/:id", userCertCtrl.GetByID)
	uc.Post("/", userCertCtrl.Create)  // terbitkan/approve
	uc.Put("/:id", userCertCtrl.Update)
	// (tambahkan Delete jika controllernya ada)
}
