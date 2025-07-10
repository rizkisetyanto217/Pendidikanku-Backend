package route

import (
	"masjidku_backend/internals/features/masjids/events/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AllEventRoutes(api fiber.Router, db *gorm.DB) {
	// 🔹 Events (user hanya melihat)
	eventCtrl := controller.NewEventController(db)
	event := api.Group("/events")
	event.Get("/", eventCtrl.GetAllEvents)
	event.Post("/by-masjid", eventCtrl.GetEventsByMasjid)
	event.Get("/:slug", eventCtrl.GetEventBySlug) // 🔥 tambahkan ini

	// 🔹 Event Sessions (user lihat jadwal sesi)
	sessionCtrl := controller.NewEventSessionController(db)
	session := api.Group("/event-sessions")
	session.Get("/all", sessionCtrl.GetAllEventSessions)
	session.Get("/by-event/:event_id", sessionCtrl.GetEventSessionsByEvent)
	session.Get("/upcoming/masjid-id/:masjid_id", sessionCtrl.GetUpcomingEventSessions)

	// 🔹 User Event Registrations
	registrationCtrl := controller.NewUserEventRegistrationController(db)
	reg := api.Group("/user-event-registrations")
	reg.Post("/", registrationCtrl.CreateRegistration)           // user daftar event
	reg.Post("/by-user", registrationCtrl.GetRegistrantsByEvent) // user lihat event yang diikuti
}
