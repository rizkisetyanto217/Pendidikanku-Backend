package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	urlscontroller "masjidku_backend/internals/features/school/submissions_assesments/submissions/controller"
)

/*
Catatan:
- Controller ini belum enforce tenant/ownership; pastikan pasang middleware:
  - RequireAuthUser: user login
  - OnlySubmissionOwner: memastikan submission-urls yg diakses milik user tsb
- Bila perlu, buat varian controller khusus user; namun untuk sekarang cukup guard di middleware.
*/

// RegisterSubmissionUrlsUserRoute
// Base: /api/u/submission-urls
// Nested opsional: /api/u/submissions/:submission_id/urls
func SubmissionUserRoutes(r fiber.Router, db *gorm.DB) {
	ctrl := urlscontroller.NewSubmissionUrlsController(db)

	// flat
	g := r.Group("/submission-urls")
	g.Post("/", ctrl.Create)     // buat URL milik submission user
	g.Get("/list", ctrl.List)        // list dengan filter ?submission_id= milik user
	g.Get("/:id", ctrl.GetByID)  // detail (owner-only)
	g.Patch("/:id", ctrl.Update) // edit (owner-only)
	g.Delete("/:id", ctrl.Delete)

	// nested (lebih ergonomis dari halaman detail submission)
	gn := r.Group("/:submission_id/urls")
	gn.Post("/", ctrl.Create)
	gn.Get("/", ctrl.List)

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
