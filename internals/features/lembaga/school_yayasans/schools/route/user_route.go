// file: internals/features/schools/schools/route/admin_dkm_route.go
package route

import (
	"schoolku_backend/internals/features/lembaga/school_yayasans/schools/controller"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func SchoolUserRoutes(admin fiber.Router, db *gorm.DB) {
	schoolCtrl := controller.NewSchoolController(db, validator.New(), nil)

	// =========================
	// 🕌 MASJID
	// =========================

	// Prefix: /schools
	schools := admin.Group("/schools")

	// OWNER-only untuk aksi sensitif/lintas tenant → /api/a/schools/owner/...
	schoolsOwner := schools.Group("/user")
	schoolsOwner.Post("/", schoolCtrl.CreateSchoolDKM)

}
