package route

import (
	ctr "masjidku_backend/internals/features/school/submissions_assesments/assesments/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Register TEACHER routes for assessment types, assessments, and assessment urls
func AssessmentTeacherRoutes(r fiber.Router, db *gorm.DB) {
	typeCtrl := ctr.NewAssessmentTypeController(db)
	assessCtrl := ctr.NewAssessmentController(db)
	urlsCtrl := ctr.NewAssessmentUrlsController(db)

	// ---------- Assessment Types (TEACHER: read-only) ----------
	typeGroup := r.Group("/assessment-types")
	typeGroup.Get("/list", typeCtrl.List)    // ?active=&q=&limit=&offset=&sort_by=&sort_dir=
	typeGroup.Get("/:id", typeCtrl.GetByID)  // detail

	// ---------- Assessments (TEACHER: manage own masjid scope) ----------
	assessGroup := r.Group("/assessments")
	assessGroup.Get("/list", assessCtrl.List)       // list + filter
	assessGroup.Get("/:id", assessCtrl.GetByID)     // detail
	assessGroup.Post("/", assessCtrl.Create)        // create
	assessGroup.Patch("/:id", assessCtrl.Patch)       // partial update (PUT-as-PATCH)
	assessGroup.Patch("/:id", assessCtrl.Patch)     // partial update (PATCH)
	assessGroup.Delete("/:id", assessCtrl.Delete)   // soft delete

	// ---------- Assessment URLs (TEACHER) ----------
	urlGroup := r.Group("/assessment-urls")
	urlGroup.Get("/list", urlsCtrl.List)            // ?assessment_id=&q=&is_published=&is_active=&page=&per_page=
	urlGroup.Get("/:id", urlsCtrl.GetByID)
	urlGroup.Post("/", urlsCtrl.Create)
	urlGroup.Patch("/:id", urlsCtrl.Update)           // PUT-as-PATCH (controller Update = patch-like)
	urlGroup.Patch("/:id", urlsCtrl.Update)         // PATCH
	urlGroup.Delete("/:id", urlsCtrl.Delete)

	// Nested endpoints (opsional, akses per assessment)
	r.Post("/assessments/:assessment_id/urls", urlsCtrl.Create)
	r.Get("/assessments/:assessment_id/urls", urlsCtrl.List)
}
