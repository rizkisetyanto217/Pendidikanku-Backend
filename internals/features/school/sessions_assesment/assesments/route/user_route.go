package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	ctr "masjidku_backend/internals/features/school/sessions_assesment/assesments/controller"
)

// AssessmentUserRoutes mendaftarkan route user untuk assessment types
// AssessmentUserRoutes mendaftarkan route USER (read-only)

// AssessmentUserRoutes mendaftarkan route USER (read-only)
func AssessmentUserRoutes(r fiber.Router, db *gorm.DB) {
	typeCtrl := ctr.NewAssessmentTypeController(db)
	assessCtrl := ctr.NewAssessmentController(db)
	urlsCtrl := ctr.NewAssessmentUrlsController(db)

	// ---------- Assessment Types (USER: read-only) ----------
	typeGroup := r.Group("/assessment-types")
	typeGroup.Get("/", typeCtrl.List)
	typeGroup.Get("/:id", typeCtrl.GetByID)

	// ---------- Assessments (USER: read-only) ----------
	assessGroup := r.Group("/assessments")
	assessGroup.Get("/", assessCtrl.List)
	assessGroup.Get("/:id", assessCtrl.GetByID)

	// ---------- Assessment URLs (USER: read-only) ----------
	urlGroup := r.Group("/assessment-urls")
	urlGroup.Get("/", urlsCtrl.List)
	urlGroup.Get("/:id", urlsCtrl.GetByID)

	// Nested list per assessment
	r.Get("/assessments/:assessment_id/urls", urlsCtrl.List)
}
