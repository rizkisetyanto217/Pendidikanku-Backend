package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	urlscontroller "masjidku_backend/internals/features/school/submissions_assesments/submissions/controller"
)

// Middleware alias biar ringkas
type Middleware = fiber.Handler

// RegisterSubmissionRoutes
// Base:
//   - /api/a/submission-urls (flat)
//   - /api/a/submissions/:submission_id/urls (nested untuk URLs)
//   - /api/a/submissions (CRUD submissions)
func SubmissionAdminRoutes(r fiber.Router, db *gorm.DB) {
	// Controller untuk URLs
	urlsCtrl := urlscontroller.NewSubmissionUrlsController(db)

	// flat URLs
	urlFlat := r.Group("/submission-urls")
	urlFlat.Post("/", urlsCtrl.Create)
	urlFlat.Get("/list", urlsCtrl.List)        // ?submission_id=&q=&is_active=&page=&per_page=
	urlFlat.Get("/:id", urlsCtrl.GetByID)  // detail URL
	urlFlat.Patch("/:id", urlsCtrl.Update) // update
	urlFlat.Delete("/:id", urlsCtrl.Delete)

	// nested URLs (lebih natural: /submissions/:submission_id/urls)
	urlNested := r.Group("/submissions/:submission_id/urls")
	urlNested.Post("/", urlsCtrl.Create)
	urlNested.Get("/", urlsCtrl.List)

	// Controller untuk Submissions
	subCtrl := urlscontroller.NewSubmissionController(db)

	sub := r.Group("/submissions")
	sub.Get("/list", subCtrl.List)             // GET   /submissions
	sub.Get("/:id", subCtrl.GetByID)       // GET   /submissions/:id
	sub.Post("/", subCtrl.Create)          // POST  /submissions
	sub.Patch("/:id", subCtrl.Patch)       // PATCH /submissions/:id
	sub.Patch("/:id/grade", subCtrl.Grade) // PATCH /submissions/:id/grade
	sub.Delete("/:id", subCtrl.Delete)     // DELETE /submissions/:id
}
