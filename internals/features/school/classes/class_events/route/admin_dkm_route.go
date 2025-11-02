// file: internals/features/school/sessions/events/routes/events_routes.go
package routes

import (
	evCtl "schoolku_backend/internals/features/school/classes/class_events/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// =========================
// ADMIN routes (auth di mount atas /admin)
// /admin/:school_id/events
// =========================
func EventAdminRoutes(r fiber.Router, db *gorm.DB) {
	// EVENTS (existing)
	ev := evCtl.NewClassEvents(db, nil)
	grp := r.Group("/:school_id/events")
	grp.Post("/", ev.Create)
	grp.Patch("/:id", ev.Patch)
	grp.Delete("/:id", ev.Delete)

	// THEMES (baru)
	th := evCtl.NewClassEventThemeController(db)
	tgrp := r.Group("/:school_id/event-themes")
	tgrp.Post("/", th.Create)
	tgrp.Patch("/:id", th.Patch)
	tgrp.Delete("/:id", th.Delete)
	tgrp.Post(":upsert", th.Upsert) // /admin/:school_id/events/themes:upsert
}
