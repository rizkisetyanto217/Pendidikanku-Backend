package route

import (
	ctr "schoolku_backend/internals/features/school/submissions_assesments/assesments/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Register USER routes for assessment types (read-only) dan assessments (listing/filter)
func AssessmentUserRoutes(r fiber.Router, db *gorm.DB) {
	typeCtrl := ctr.NewAssessmentTypeController(db)
	assessCtrl := ctr.NewAssessmentController(db)

	// TANPA :school_id – pakai ResolveSchoolContext / token
	g := r.Group("")

	// ---------- Assessment Types (USER/TEACHER: read-only) ----------
	typeGroup := g.Group("/assessment-types")
	typeGroup.Get("/list", typeCtrl.List) // ?active=&q=&limit=&offset=&sort_by=&sort_dir=

	// ---------- Assessments (USER/TEACHER: list + filter) ----------
	assessGroup := g.Group("/assessments")
	assessGroup.Get("/list", assessCtrl.List) // list + filter
}
