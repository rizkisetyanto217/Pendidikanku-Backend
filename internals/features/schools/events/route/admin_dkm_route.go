package route

import (
	"schoolku_backend/internals/constants"
	"schoolku_backend/internals/features/schools/events/controller"
	authMiddleware "schoolku_backend/internals/middlewares/auth"
	schoolkuMiddleware "schoolku_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Login wajib + role admin/dkm/owner + scope school
func EventAdminRoutes(api fiber.Router, db *gorm.DB) {
	admin := api.Group("/",
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola event"),
			constants.AdminAndAbove, // admin, dkm, owner
		),
		schoolkuMiddleware.IsSchoolAdmin(), // inject/cek school_id dari token
	)

	// ---------- Events (CUD + internal read by school) ----------
	eventCtrl := controller.NewEventController(db)
	events := admin.Group("/events")
	events.Post("/", eventCtrl.CreateEvent)
	events.Patch("/:id", eventCtrl.UpdateEvent) // validasi :id di controller
	events.Delete("/:id", eventCtrl.DeleteEvent)
	events.Get("/by-school", eventCtrl.GetEventsBySchool)

	// ---------- Event Sessions (CUD + admin read by event) ----------
	sessionCtrl := controller.NewEventSessionController(db)
	sessions := admin.Group("/event-sessions")
	sessions.Post("/", sessionCtrl.CreateEventSession)
	sessions.Put("/:id", sessionCtrl.UpdateEventSession)                     // validasi :id di controller
	sessions.Delete("/:id", sessionCtrl.DeleteEventSession)                  // validasi :id di controller
	sessions.Get("/by-event/:event_id", sessionCtrl.GetEventSessionsByEvent) // validasi :event_id di controller

	// ---------- Registrants (laporan internal) ----------
	regCtrl := controller.NewUserEventRegistrationController(db)
	reg := admin.Group("/user-event-registrations")
	reg.Post("/by-event", regCtrl.GetRegistrantsByEvent)
}
