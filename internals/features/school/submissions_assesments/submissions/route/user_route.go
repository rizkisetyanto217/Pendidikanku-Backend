package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	submissionController "schoolku_backend/internals/features/school/submissions_assesments/submissions/controller"
)

// RegisterSubmissionUserRoute
// Base: /api/u/:school_id/submissions
func SubmissionUserRoutes(r fiber.Router, db *gorm.DB) {
	// Controller untuk Submissions
	subCtrl := submissionController.NewSubmissionController(db)

	// Group by school_id di path
	g := r.Group("/:school_id")

	sub := g.Group("/submissions")
	sub.Get("/list", subCtrl.List)     // GET    /:school_id/submissions/list
	sub.Post("/", subCtrl.Create)      // POST   /:school_id/submissions
	sub.Patch("/:id", subCtrl.Patch)   // PATCH  /:school_id/submissions/:id
	sub.Delete("/:id", subCtrl.Delete) // DELETE /:school_id/submissions/:id
}
