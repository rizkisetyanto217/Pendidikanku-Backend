package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	submissionController "schoolku_backend/internals/features/school/submissions_assesments/submissions/controller"
)

// Middleware alias biar ringkas
type Middleware = fiber.Handler

// RegisterSubmissionRoutes (ADMIN / STAFF)
// Base (di /api/a misalnya):
//   - GET    /api/a/submissions/list
//   - POST   /api/a/submissions
//   - PATCH  /api/a/submissions/:id/urls
//   - DELETE /api/a/submissions/:id/urls/:urlId
// School diambil dari token (active school), bukan dari path.
func SubmissionAdminRoutes(r fiber.Router, db *gorm.DB) {
	// Controller untuk Submissions
	subCtrl := submissionController.NewSubmissionController(db)

	// Tanpa :school_id di path
	sub := r.Group("/submissions")

	// LIST
	sub.Get("/list", subCtrl.List) // GET    /submissions/list

	// CREATE (boleh juga dipakai staff kalau controller mengizinkan)
	sub.Post("/", subCtrl.Create) // POST   /submissions

	// UPDATE URLS (nested)
	sub.Patch("/:id/urls", subCtrl.Patch) // PATCH  /submissions/:id/urls

	// DELETE URL
	sub.Delete("/:id/urls/:urlId", subCtrl.Delete) // DELETE /submissions/:id/urls/:urlId
}