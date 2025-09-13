package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	ctr "masjidku_backend/internals/features/school/submissions_assesments/assesments/controller"
)

// RegisterAdminAssessmentTypeRoutes mendaftarkan route admin untuk assessment types
// AssessmentAdminRoutes mendaftarkan route ADMIN untuk assessment types & assessments

// AssessmentAdminRoutes mendaftarkan route ADMIN untuk assessment types, assessments, dan assessment urls
func AssessmentAdminRoutes(r fiber.Router, db *gorm.DB) {
	typeCtrl := ctr.NewAssessmentTypeController(db)
	assessCtrl := ctr.NewAssessmentController(db)

	// ---------- Assessment Types (ADMIN: full CRUD) ----------
	typeGroup := r.Group("/assessment-types")
	typeGroup.Post("/", typeCtrl.Create)       // create
	typeGroup.Patch("/:id", typeCtrl.Patch)     // partial update (PUT-as-PATCH)
	typeGroup.Delete("/:id", typeCtrl.Delete)  // soft delete

	// ---------- Assessments (ADMIN) ----------
	assessGroup := r.Group("/assessments")
	assessGroup.Post("/", assessCtrl.Create)      // create
	assessGroup.Put("/:id", assessCtrl.Patch)    // partial update (PUT-as-PATCH)
	assessGroup.Patch("/:id", assessCtrl.Patch)  // partial update (PATCH)
	assessGroup.Delete("/:id", assessCtrl.Delete) // soft delete

}