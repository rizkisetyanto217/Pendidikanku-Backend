// file: internals/features/school/schedule_rooms/rooms/routes/class_room_user_routes.go
package routes

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"masjidku_backend/internals/features/school/academics/rooms/controller"
)

// RoomsUserRoutes — route USER (read-only).
// Contoh mount dari caller:
//   user := api.Group("/api/u") // atau "/api/p" jika publik
//   routes.RoomsUserRoutes(user, db)
func RoomsUserRoutes(user fiber.Router, db *gorm.DB) {
	ctl := controller.NewClassRoomController(db, nil) // validator nil
	g := user.Group("/class-rooms")

	// Read-only endpoints
	g.Get("/list", ctl.List)

	ctl2 := controller.NewClassRoomVirtualLinkController(db)
	h := user.Group("/class-room-virtual-links")

	// Read-only endpoints
	h.Get("/", ctl2.List)      // e.g. /api/u/class-room-virtual-links
	h.Get("/list", ctl2.List)  // alias kalau mau konsisten dengan pola /list
	h.Get("/:id", ctl2.Detail) // optional detail
}
