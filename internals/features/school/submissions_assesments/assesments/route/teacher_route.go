package route

import (
	ctr "schoolku_backend/internals/features/school/submissions_assesments/assesments/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Register TEACHER routes for assessment types, assessments, and assessment urls
func AssessmentTeacherRoutes(r fiber.Router, db *gorm.DB) {
	assessCtrl := ctr.NewAssessmentController(db)

	// Base group pakai :school_id di path
	g := r.Group("/:school_id")

	// ---------- Assessments (TEACHER: manage own school scope) ----------
	assessGroup := g.Group("/assessments")
	assessGroup.Post("/", assessCtrl.Create)      // create
	assessGroup.Put("/:id", assessCtrl.Patch)     // partial update (PUT-as-PATCH)
	assessGroup.Patch("/:id", assessCtrl.Patch)   // partial update (PATCH)
	assessGroup.Delete("/:id", assessCtrl.Delete) // soft delete
}
