package route

import (
	"masjidku_backend/internals/constants"
	"masjidku_backend/internals/features/masjids/events/controller"
	authMiddleware "masjidku_backend/internals/middlewares/auth"
	masjidkuMiddleware "masjidku_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Login wajib + role admin/dkm/owner + scope masjid
func EventAdminRoutes(api fiber.Router, db *gorm.DB) {
	admin := api.Group("/",
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola event"),
			constants.AdminAndAbove, // admin, dkm, owner
		),
		masjidkuMiddleware.IsMasjidAdmin(), // inject/cek masjid_id dari token
	)

	// ---------- Events (CUD + internal read by masjid) ----------
	eventCtrl := controller.NewEventController(db)
	events := admin.Group("/events")
	events.Post("/", eventCtrl.CreateEvent)
	events.Patch("/:id", eventCtrl.UpdateEvent) // validasi :id di controller
	events.Delete("/:id", eventCtrl.DeleteEvent)
	events.Get("/by-masjid", eventCtrl.GetEventsByMasjid)

	// ---------- Event Sessions (CUD + admin read by event) ----------
	sessionCtrl := controller.NewEventSessionController(db)
	sessions := admin.Group("/event-sessions")
	sessions.Post("/", sessionCtrl.CreateEventSession)
	sessions.Put("/:id", sessionCtrl.UpdateEventSession)     // validasi :id di controller
	sessions.Delete("/:id", sessionCtrl.DeleteEventSession)  // validasi :id di controller
	sessions.Get("/by-event/:event_id", sessionCtrl.GetEventSessionsByEvent) // validasi :event_id di controller

	// ---------- Registrants (laporan internal) ----------
	regCtrl := controller.NewUserEventRegistrationController(db)
	reg := admin.Group("/user-event-registrations")
	reg.Post("/by-event", regCtrl.GetRegistrantsByEvent)
}
