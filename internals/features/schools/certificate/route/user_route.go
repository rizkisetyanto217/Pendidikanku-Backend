package route

import (
	"schoolku_backend/internals/constants"
	certificateController "schoolku_backend/internals/features/schools/certificate/controller"
	authMiddleware "schoolku_backend/internals/middlewares/auth"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// User login: kelola/lihat sertifikat milik sendiri
func CertificateUserRoutes(api fiber.Router, db *gorm.DB) {
	user := api.Group("/",
		authMiddleware.OnlyRolesSlice(
			"❌ Hanya pengguna terautentikasi yang boleh mengakses sertifikat.",
			constants.AllowedRoles,
		),
	)

	userCertCtrl := certificateController.NewUserCertificateController(db)
	uc := user.Group("/user-certificates")
	uc.Get("/", userCertCtrl.GetAll)     // list milik user
	uc.Get("/:id", userCertCtrl.GetByID) // detail milik user
	uc.Post("/", userCertCtrl.Create)    // klaim/ajukan/terbitkan (sesuai logic controller)
	uc.Put("/:id", userCertCtrl.Update)  // update milik user (mis. nama pada sertifikat)
}
