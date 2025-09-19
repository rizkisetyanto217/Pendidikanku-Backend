package route

import (
	ctr "masjidku_backend/internals/features/school/submissions_assesments/assesments/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Register TEACHER routes for assessment types, assessments, and assessment urls
func AssessmentTeacherRoutes(r fiber.Router, db *gorm.DB) {
	assessCtrl := ctr.NewAssessmentController(db)

	// Base group pakai :masjid_id di path
	g := r.Group("/:masjid_id")

	// ---------- Assessments (TEACHER: manage own masjid scope) ----------
	assessGroup := g.Group("/assessments")
	assessGroup.Post("/", assessCtrl.Create)      // create
	assessGroup.Put("/:id", assessCtrl.Patch)     // partial update (PUT-as-PATCH)
	assessGroup.Patch("/:id", assessCtrl.Patch)   // partial update (PATCH)
	assessGroup.Delete("/:id", assessCtrl.Delete) // soft delete
}
