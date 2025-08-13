package route

import (
	"masjidku_backend/internals/features/masjids/events/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func EventRoutes(api fiber.Router, db *gorm.DB) {
	// 🔹 Events
	eventCtrl := controller.NewEventController(db)
	event := api.Group("/events")
	event.Post("/", eventCtrl.CreateEvent)
	event.Get("/:id", eventCtrl.GetEventByID)
	event.Patch("/:id", eventCtrl.UpdateEvent)
	// event.Post("/by-masjid", eventCtrl.GetEventsByMasjid)

	// 🔹 Event Sessions (admin can create)
	sessionCtrl := controller.NewEventSessionController(db)
	session := api.Group("/event-sessions")
	session.Post("/", sessionCtrl.CreateEventSession)
	session.Get("/:event_id", sessionCtrl.GetEventSessionsByEvent)
	session.Get("/all", sessionCtrl.GetAllEventSessions)
	session.Get("/upcoming", sessionCtrl.GetUpcomingEventSessions)

	// 🔹 User Event Registrations
	registrationCtrl := controller.NewUserEventRegistrationController(db)
	reg := api.Group("/user-event-registrations")
	reg.Post("/", registrationCtrl.CreateRegistration)
	reg.Post("/by-event", registrationCtrl.GetRegistrantsByEvent)
}
