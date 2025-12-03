// file: internals/features/schools/schools/route/public_route.go (atau sesuai nama file kamu)
package route

import (
	"madinahsalam_backend/internals/features/lembaga/school_yayasans/schools/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func LembagaUserRoutes(user fiber.Router, db *gorm.DB) {
	schoolCtrl := controller.NewSchoolController(db, nil, nil)

	// 🕌 Group: /schools
	school := user.Group("/schools")

	// ⬅️ ini yang tadinya salah
	school.Post("/", schoolCtrl.CreateSchoolDKM)

}
