package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	ctr "madinahsalam_backend/internals/features/school/submissions_assesments/assesments/controller"
)

// AssessmentAdminRoutes mendaftarkan route ADMIN untuk assessments
// Sekarang TIDAK scoped lagi pakai :school_id di path, tapi pakai school context (token/slug).
func AssessmentAdminRoutes(r fiber.Router, db *gorm.DB) {
	typeCtrl := ctr.NewAssessmentTypeController(db)
	assessCtrl := ctr.NewAssessmentController(db)

	// Tanpa :school_id
	g := r.Group("")

	// ---------- Assessment Types (DKM/Admin, via school context) ----------
	typeGroup := g.Group("/assessment-types")
	typeGroup.Post("/", typeCtrl.Create)      // create
	typeGroup.Patch("/:id", typeCtrl.Patch)   // partial update
	typeGroup.Delete("/:id", typeCtrl.Delete) // soft delete

	// ---------- Assessments (DKM/Teacher, via school context) ----------
	assessGroup := g.Group("/assessments")
	assessGroup.Post("/", assessCtrl.Create)      // create
	assessGroup.Put("/:id", assessCtrl.Patch)     // partial update (PUT-as-PATCH)
	assessGroup.Patch("/:id", assessCtrl.Patch)   // partial update
	assessGroup.Delete("/:id", assessCtrl.Delete) // soft delete
}
