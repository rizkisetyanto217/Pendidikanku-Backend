package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	urlscontroller "masjidku_backend/internals/features/school/sessions_assesment/submissions/controller"
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
func RegisterSubmissionUrlsUserRoute(app *fiber.App, db *gorm.DB, middlewares ...fiber.Handler) {
	ctrl := urlscontroller.NewSubmissionUrlsController(db)

	// flat
	g := app.Group("/submission-urls", middlewares...)
	g.Post("/", ctrl.Create)     // buat URL milik submission user
	g.Get("/", ctrl.List)        // list dengan filter ?submission_id= milik user
	g.Get("/:id", ctrl.GetByID)  // detail (owner-only)
	g.Patch("/:id", ctrl.Update) // edit (owner-only)
	g.Delete("/:id", ctrl.Delete)

	// nested (lebih ergonomis dari halaman detail submission)
	gn := app.Group("/:submission_id/urls", middlewares...)
	gn.Post("/", ctrl.Create)
	gn.Get("/", ctrl.List)
}
