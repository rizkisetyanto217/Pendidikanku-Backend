// file: internals/features/schools/schools/route/public_route.go (atau sesuai nama file kamu)
package route

import (
	"schoolku_backend/internals/features/lembaga/school_yayasans/schools/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AllSchoolRoutes(user fiber.Router, db *gorm.DB) {
	schoolCtrl := controller.NewSchoolController(db, nil, nil)
	profileCtrl := controller.NewSchoolProfileController(db, nil)
	plan := controller.NewSchoolServicePlanController(db, nil)

	// 🕌 Group: /schools
	school := user.Group("/schools")

	// Lebih spesifik dulu supaya tidak bentrok dengan "/:slug"
	school.Get("/verified", schoolCtrl.GetAllVerifiedSchools)
	school.Get("/verified/:id", schoolCtrl.GetVerifiedSchoolByID)

	school.Get("/list", schoolCtrl.GetAllSchools)    // 📄 Semua school
	school.Get("/:slug", schoolCtrl.GetSchoolBySlug) // 🔍 Detail by slug

	// 📄 Group: /school-profiles
	profile := user.Group("/school-profiles")

	// Read-only endpoints yang tersedia di controller
	profile.Get("/list", profileCtrl.List) // list + filter + pagination

	// alias lama (opsional):
	alias := user.Group("/school-service-plans")
	alias.Get("/list", plan.List)

}
