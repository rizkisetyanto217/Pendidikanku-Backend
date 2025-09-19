package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	ctr "masjidku_backend/internals/features/school/submissions_assesments/assesments/controller"
)

// AssessmentAdminRoutes mendaftarkan route ADMIN untuk assessment types, assessments (scoped by :masjid_id)
func AssessmentAdminRoutes(r fiber.Router, db *gorm.DB) {
	typeCtrl := ctr.NewAssessmentTypeController(db)
	assessCtrl := ctr.NewAssessmentController(db)

	// Base group pakai :masjid_id di path
	g := r.Group("/:masjid_id")

	// ---------- Assessment Types (ADMIN: full CRUD) ----------
	typeGroup := g.Group("/assessment-types")
	typeGroup.Post("/", typeCtrl.Create)      // create
	typeGroup.Patch("/:id", typeCtrl.Patch)   // partial update (PATCH)
	typeGroup.Delete("/:id", typeCtrl.Delete) // soft delete

	// ---------- Assessments (ADMIN) ----------
	assessGroup := g.Group("/assessments")
	assessGroup.Post("/", assessCtrl.Create)      // create
	assessGroup.Put("/:id", assessCtrl.Patch)     // partial update (PUT-as-PATCH)
	assessGroup.Patch("/:id", assessCtrl.Patch)   // partial update (PATCH)
	assessGroup.Delete("/:id", assessCtrl.Delete) // soft delete
}
