package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	submissionController "madinahsalam_backend/internals/features/school/submissions_assesments/submissions/controller"
)

// RegisterSubmissionUserRoutes
// Base (di /api/u misalnya):
//   - GET    /api/u/submissions/list
//   - POST   /api/u/submissions
//   - PATCH  /api/u/submissions/:id/urls
//   - DELETE /api/u/submissions/:id/urls/:urlId
//
// Sama-sama ambil school dari token; guard role di dalam controller/helper.
func SubmissionUserRoutes(r fiber.Router, db *gorm.DB) {
	subCtrl := submissionController.NewSubmissionController(db)

	// Tanpa :school_id di path
	sub := r.Group("/submissions")

	sub.Get("/list", subCtrl.List)                 // GET    /submissions/list
	sub.Post("/", subCtrl.Create)                  // POST   /submissions
	sub.Patch("/:id/urls", subCtrl.Patch)          // PATCH  /submissions/:id/urls
	sub.Delete("/:id/urls/:urlId", subCtrl.Delete) // DELETE /submissions/:id/urls/:urlId
}
