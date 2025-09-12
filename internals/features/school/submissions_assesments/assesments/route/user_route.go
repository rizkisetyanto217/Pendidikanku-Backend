package route

import (
	ctr "masjidku_backend/internals/features/school/submissions_assesments/assesments/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	// ctr "masjidku_backend/internals/features/school/submissions_assesments/assesments/controller"
)

// Register TEACHER routes for assessment types, assessments, and assessment urls
func AssessmentUserRoutes(r fiber.Router, db *gorm.DB) {
	typeCtrl := ctr.NewAssessmentTypeController(db)
	assessCtrl := ctr.NewAssessmentController(db)
	urlsCtrl := ctr.NewAssessmentUrlsController(db)

	// ---------- Assessment Types (TEACHER: read-only) ----------
	typeGroup := r.Group("/assessment-types")
	typeGroup.Get("/list", typeCtrl.List)    // ?active=&q=&limit=&offset=&sort_by=&sort_dir=

	// ---------- Assessments (TEACHER: manage own masjid scope) ----------
	assessGroup := r.Group("/assessments")
	assessGroup.Get("/list", assessCtrl.List)       // list + filter

	// ---------- Assessment URLs (TEACHER) ----------
	urlGroup := r.Group("/assessment-urls")
	urlGroup.Get("/list", urlsCtrl.List)            // ?assessment_id=&q=&is_published=&is_active=&page=&per_page=

	r.Get("/assessments/:assessment_id/urls", urlsCtrl.List)
}
