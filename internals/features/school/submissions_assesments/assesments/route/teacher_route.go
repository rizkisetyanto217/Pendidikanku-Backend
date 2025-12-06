package route

import (
	assesmentsController "madinahsalam_backend/internals/features/school/submissions_assesments/assesments/controller/assesments"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Register TEACHER routes for assessments (manage own school scope via token/context)
func AssessmentTeacherRoutes(r fiber.Router, db *gorm.DB) {
	assessCtrl := assesmentsController.NewAssessmentController(db)

	// TANPA :school_id – school diambil dari context/token di controller
	g := r.Group("")

	// ---------- Assessments (TEACHER: manage own school scope) ----------
	assessGroup := g.Group("/assessments")
	assessGroup.Post("/", assessCtrl.Create)      // create
	assessGroup.Put("/:id", assessCtrl.Patch)     // partial update (PUT-as-PATCH)
	assessGroup.Patch("/:id", assessCtrl.Patch)   // partial update (PATCH)
	assessGroup.Delete("/:id", assessCtrl.Delete) // soft delete
}
