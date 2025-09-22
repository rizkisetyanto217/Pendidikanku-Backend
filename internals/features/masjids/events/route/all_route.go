package route

import (
	"masjidku_backend/internals/features/masjids/events/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Tidak perlu login
func AllEventRoutes(api fiber.Router, db *gorm.DB) {
	// Events (public read)
	eventCtrl := controller.NewEventController(db)
	events := api.Group("/events")
	events.Get("/", eventCtrl.GetAllEvents)
	events.Get("/by-masjid-slug/:slug", eventCtrl.GetEventsByMasjidSlug)
	events.Get("/:slug", eventCtrl.GetEventBySlug)

	// Event Sessions (public read)
	sessionCtrl := controller.NewEventSessionController(db)
	sessions := api.Group("/event-sessions")
	sessions.Get("/", sessionCtrl.GetAllEventSessions)
	// upcoming publik (opsional filter masjid_id di path)
	// contoh: /event-sessions/upcoming/masjid_id/<uuid>, atau kosongkan masjid_id untuk semua masjid publik
	sessions.Get("/upcoming/masjid_id/:masjid_id", sessionCtrl.GetUpcomingEventSessions)
}
