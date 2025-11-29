// file: internals/features/schools/schools/route/public_route.go (atau sesuai nama file kamu)
package route

import (
	"madinahsalam_backend/internals/features/lembaga/school_yayasans/schools/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AllLembagaRoutes(user fiber.Router, db *gorm.DB) {
	schoolCtrl := controller.NewSchoolController(db, nil, nil)
	profileCtrl := controller.NewSchoolProfileController(db, nil)
	plan := controller.NewSchoolServicePlanController(db, nil)

	// 🕌 Group: /schools
	school := user.Group("/schools")

	// ⬅️ ini yang tadinya salah
	school.Post("/", schoolCtrl.CreateSchoolDKM)

	// Lebih spesifik dulu supaya tidak bentrok dengan "/:slug"
	school.Get("/list", schoolCtrl.GetSchools)       // 📄 Semua school

	// 📄 Group: /school-profiles
	profile := user.Group("/school-profiles")

	// Read-only endpoints yang tersedia di controller
	profile.Get("/list", profileCtrl.List) // list + filter + pagination

	// alias lama (opsional):
	alias := user.Group("/school-service-plans")
	alias.Get("/list", plan.List)

}
