package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	urlscontroller "masjidku_backend/internals/features/school/submissions_assesment/submissions/controller"
)

// Middleware alias biar ringkas
type Middleware = fiber.Handler

// RegisterSubmissionUrlsAdminRoute
// Base: /api/a/submission-urls
// Nested opsional: /api/a/submissions/:submission_id/urls
func RegisterSubmissionUrlsAdminRoute(app *fiber.App, db *gorm.DB, middlewares ...Middleware) {
	ctrl := urlscontroller.NewSubmissionUrlsController(db)

	// flat
	g := app.Group("/submission-urls", middlewares...)
	g.Post("/", ctrl.Create)
	g.Get("/", ctrl.List)        // ?submission_id=&q=&is_active=&page=&per_page=
	g.Get("/:id", ctrl.GetByID)
	g.Patch("/:id", ctrl.Update)
	g.Delete("/:id", ctrl.Delete)

	// nested (opsional, memanfaatkan path param submission_id di controller)
	gn := app.Group("/:submission_id/urls", middlewares...)
	gn.Post("/", ctrl.Create) // create URL untuk submission tertentu
	gn.Get("/", ctrl.List)    // list URL milik submission tertentu
}
