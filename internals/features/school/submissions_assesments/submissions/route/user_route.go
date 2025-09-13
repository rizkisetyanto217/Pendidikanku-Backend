package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	urlscontroller "masjidku_backend/internals/features/school/submissions_assesments/submissions/controller"
)


// RegisterSubmissionUrlsUserRoute
// Base: /api/u/submission-urls
// Nested opsional: /api/u/submissions/:submission_id/urls
func SubmissionUserRoutes(r fiber.Router, db *gorm.DB) {

		// Controller untuk Submissions
	subCtrl := urlscontroller.NewSubmissionController(db)

	sub := r.Group("/submissions")
	sub.Get("/list", subCtrl.List)             // GET   /submissions
	sub.Post("/", subCtrl.Create)          // POST  /submissions
	sub.Patch("/:id", subCtrl.Patch)       // PATCH /submissions/:id
	sub.Patch("/:id/grade", subCtrl.Grade) // PATCH /submissions/:id/grade
	sub.Delete("/:id", subCtrl.Delete)     // DELETE /submissions/:id
}
