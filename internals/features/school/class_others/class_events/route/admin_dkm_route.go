// file: internals/features/school/sessions/events/routes/events_routes.go
package routes

import (
	eventsController "madinahsalam_backend/internals/features/school/class_others/class_events/controller/events"
	themesController "madinahsalam_backend/internals/features/school/class_others/class_events/controller/themes"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// =========================
// ADMIN routes (auth di mount atas /admin)
// /admin/:school_id/events
// =========================
func EventAdminRoutes(r fiber.Router, db *gorm.DB) {
	// EVENTS (existing)
	ev := eventsController.NewClassEvents(db, nil)
	grp := r.Group("/:school_id/events")
	grp.Post("/", ev.Create)
	grp.Patch("/:id", ev.Patch)
	grp.Delete("/:id", ev.Delete)

	// THEMES (baru)
	th := themesController.NewClassEventThemeController(db)
	tgrp := r.Group("/:school_id/event-themes")
	tgrp.Post("/", th.Create)
	tgrp.Patch("/:id", th.Patch)
	tgrp.Delete("/:id", th.Delete)
	tgrp.Post(":upsert", th.Upsert) // /admin/:school_id/events/themes:upsert
}
