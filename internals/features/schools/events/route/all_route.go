package route

import (
	"schoolku_backend/internals/features/schools/events/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Tidak perlu login
func AllEventRoutes(api fiber.Router, db *gorm.DB) {
	// Events (public read)
	eventCtrl := controller.NewEventController(db)
	events := api.Group("/events")
	events.Get("/", eventCtrl.GetAllEvents)
	events.Get("/by-school-slug/:slug", eventCtrl.GetEventsBySchoolSlug)
	events.Get("/:slug", eventCtrl.GetEventBySlug)

	// Event Sessions (public read)
	sessionCtrl := controller.NewEventSessionController(db)
	sessions := api.Group("/event-sessions")
	sessions.Get("/", sessionCtrl.GetAllEventSessions)
	// upcoming publik (opsional filter school_id di path)
	// contoh: /event-sessions/upcoming/school_id/<uuid>, atau kosongkan school_id untuk semua school publik
	sessions.Get("/upcoming/school_id/:school_id", sessionCtrl.GetUpcomingEventSessions)
}
