// file: internals/features/school/schedule_rooms/rooms/routes/class_room_user_routes.go
package routes

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	clsCtl "masjidku_backend/internals/features/school/academics/rooms/controller"
)

// RoomsUserRoutes — route USER (read-only).
// Contoh mount dari caller:
//   user := api.Group("/api/u") // atau "/api/p" jika publik
//   routes.RoomsUserRoutes(user, db)
func RoomsUserRoutes(user fiber.Router, db *gorm.DB) {
	ctl := clsCtl.NewClassRoomController(db, nil)
	// Tambah masjid_id agar ResolveMasjidContext bisa resolve dari path
	g := user.Group("/:masjid_id/class-rooms")

	// Read-only endpoints
	g.Get("/list", ctl.List)

	ctl2 := clsCtl.NewClassRoomVirtualLinkController(db)
	h := user.Group("/:masjid_id/class-room-virtual-links")

	// Read-only endpoints
	h.Get("/", ctl2.List)      // e.g. /api/u/masjids/:masjid_id/class-room-virtual-links
	h.Get("/list", ctl2.List)  // alias /list
	h.Get("/:id", ctl2.Detail) // detail
}
