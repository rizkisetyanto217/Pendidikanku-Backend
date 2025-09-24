// file: internals/features/school/sessions/events/routes/events_routes.go
package routes

import (
	evCtl "masjidku_backend/internals/features/school/others/events/controller"
	// ← import tambahan
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// =========================
// USER / PUBLIC routes (read-only)
// /:masjid_id/events
// =========================
func AllEventRoutes(r fiber.Router, db *gorm.DB) {
	// Events
	ev := evCtl.NewClassEvents(db, nil)
	grp := r.Group("/:masjid_id/events")
	grp.Get("/", ev.List)
	grp.Get("/list", ev.List)

	// Event Themes
	th := evCtl.NewClassEventThemeController(db)
	tgrp := r.Group("/:masjid_id/event-themes")
	tgrp.Get("/", th.List)
	tgrp.Get("/list", th.List)
}
