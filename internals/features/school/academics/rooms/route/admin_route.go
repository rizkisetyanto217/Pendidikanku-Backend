// file: internals/features/school/schedule_rooms/rooms/routes/class_room_admin_routes.go
package routes

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	clsCtl "masjidku_backend/internals/features/school/academics/rooms/controller"
)

// RoomsAdminRoutes — route khusus ADMIN (CRUD penuh + restore).
// Contoh mount dari caller:
//   admin := api.Group("/api/a") // atau sesuai prefix kamu
//   routes.RoomsAdminRoutes(admin, db)
func RoomsAdminRoutes(admin fiber.Router, db *gorm.DB) {
	ctl := clsCtl.NewClassRoomController(db, nil) // validator nil
	// Tambah :masjid_id biar ResolveMasjidContext bisa resolve dari path
	g := admin.Group("/:masjid_id/class-rooms")

	// Write
	g.Post("/", ctl.Create)
	g.Patch("/:id", ctl.Patch)

	// Soft delete + restore
	g.Delete("/:id", ctl.Delete)
	g.Post("/:id/restore", ctl.Restore)

	// Virtual Links
	ctl2 := clsCtl.NewClassRoomVirtualLinkController(db)
	h := admin.Group("/:masjid_id/class-room-virtual-links")

	// Create / Read
	h.Post("/", ctl2.Create)

	// Update
	h.Patch("/:id", ctl2.Update)

	// Soft delete + restore (+ hard delete via ?hard=true)
	h.Delete("/:id", ctl2.Delete)
}
