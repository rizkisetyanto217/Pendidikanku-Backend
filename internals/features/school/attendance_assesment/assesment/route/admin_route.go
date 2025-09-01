package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	ctr "masjidku_backend/internals/features/school/attendance_assesment/assesment/controller"
)

// RegisterAdminAssessmentTypeRoutes mendaftarkan route admin untuk assessment types
// AssessmentAdminRoutes mendaftarkan route ADMIN untuk assessment types & assessments

// AssessmentAdminRoutes mendaftarkan route ADMIN untuk assessment types, assessments, dan assessment urls
func AssessmentAdminRoutes(r fiber.Router, db *gorm.DB) {
	typeCtrl := ctr.NewAssessmentTypeController(db)
	assessCtrl := ctr.NewAssessmentController(db)
	urlsCtrl := ctr.NewAssessmentUrlsController(db)

	// ---------- Assessment Types (ADMIN: full CRUD) ----------
	typeGroup := r.Group("/assessment-types")
	typeGroup.Get("/", typeCtrl.List)          // ?masjid_id=&active=&q=&limit=&offset=&sort_by=&sort_dir=
	typeGroup.Get("/:id", typeCtrl.GetByID)    // detail
	typeGroup.Post("/", typeCtrl.Create)       // create
	typeGroup.Put("/:id", typeCtrl.Update)     // partial update (PUT-as-PATCH)
	typeGroup.Delete("/:id", typeCtrl.Delete)  // soft delete

	// ---------- Assessments (ADMIN) ----------
	assessGroup := r.Group("/assessments")
	assessGroup.Get("/", assessCtrl.List)        // list + filter
	assessGroup.Get("/:id", assessCtrl.GetByID)  // detail
	assessGroup.Post("/", assessCtrl.Create)     // create
	assessGroup.Put("/:id", assessCtrl.Patch)    // partial update (PUT-as-PATCH)
	assessGroup.Patch("/:id", assessCtrl.Patch)  // partial update (PATCH)
	assessGroup.Delete("/:id", assessCtrl.Delete) // soft delete

	// ---------- Assessment URLs (ADMIN) ----------
	urlGroup := r.Group("/assessment-urls")
	urlGroup.Get("/", urlsCtrl.List)            // ?assessment_id=&q=&is_published=&is_active=&page=&per_page=
	urlGroup.Get("/:id", urlsCtrl.GetByID)
	urlGroup.Post("/", urlsCtrl.Create)
	urlGroup.Put("/:id", urlsCtrl.Update)       // PUT-as-PATCH (controller Update = patch-like)
	urlGroup.Patch("/:id", urlsCtrl.Update)     // PATCH
	urlGroup.Delete("/:id", urlsCtrl.Delete)

	// Nested endpoints (opsional, mempermudah akses per assessment)
	r.Post("/assessments/:assessment_id/urls", urlsCtrl.Create)
	r.Get("/assessments/:assessment_id/urls", urlsCtrl.List)
}