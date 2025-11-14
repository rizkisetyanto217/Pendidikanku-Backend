package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	submissionController "schoolku_backend/internals/features/school/submissions_assesments/submissions/controller"
)

// Middleware alias biar ringkas
type Middleware = fiber.Handler

// RegisterSubmissionRoutes (ADMIN)
// Base (di /api/a misalnya):
//   - /api/a/schools/:school_id/submissions          (LIST + CREATE)
//   - /api/a/schools/:school_id/submissions/:id      (PATCH, DELETE, GET BY ID kalau nanti ada)
//   - (nanti) /api/a/schools/:school_id/submissions/:submission_id/urls  → nested URLs
func SubmissionAdminRoutes(r fiber.Router, db *gorm.DB) {
	// Controller untuk Submissions
	subCtrl := submissionController.NewSubmissionController(db)

	// Group dengan school_id di path → dibaca oleh ResolveSchoolContext
	sub := r.Group("/schools/:school_id/submissions")

	// LIST
	sub.Get("/", subCtrl.List) // GET    /schools/:school_id/submissions

	// CREATE
	sub.Post("/", subCtrl.Create) // POST   /schools/:school_id/submissions

	// UPDATE
	sub.Patch("/:id", subCtrl.Patch) // PATCH  /schools/:school_id/submissions/:id

	// DELETE
	sub.Delete("/:id", subCtrl.Delete) // DELETE /schools/:school_id/submissions/:id
}
