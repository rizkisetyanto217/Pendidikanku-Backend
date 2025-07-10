package route

import (
	"masjidku_backend/internals/features/certificates/certificate_versions/controller"
	"masjidku_backend/internals/constants"
	"masjidku_backend/internals/middlewares/auth"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func CertificateVersionAdminRoutes(api fiber.Router, db *gorm.DB) {
	c := controller.NewCertificateVersionController(db)

	// 🔐 Hanya untuk role admin, teacher, owner
	protectedRoutes := api.Group("/certificate-versions",
		auth.OnlyRoles(
			constants.RoleErrorTeacher("mengelola versi sertifikat"),
			constants.AdminOnly...,
		),
	)

	protectedRoutes.Get("/", c.GetAll)
	protectedRoutes.Get("/:id", c.GetByID)
	protectedRoutes.Post("/", c.Create)
	protectedRoutes.Put("/:id", c.Update)
	protectedRoutes.Delete("/:id", c.Delete)
}
