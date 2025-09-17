// file: internals/features/school/schedule_rooms/rooms/routes/class_room_admin_routes.go
package routes

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"masjidku_backend/internals/features/school/academics/rooms/controller"
)

// RoomsAdminRoutes — route khusus ADMIN (CRUD penuh + restore).
// Contoh mount dari caller:
//   admin := api.Group("/api/a") // atau sesuai prefix kamu
//   routes.RoomsAdminRoutes(admin, db)
func RoomsAdminRoutes(admin fiber.Router, db *gorm.DB) {
	ctl := controller.NewClassRoomController(db, nil) // validator nil
	g := admin.Group("/class-rooms")

	// Write
	g.Post("/", ctl.Create)
	g.Put("/:id", ctl.Update)
	g.Patch("/:id", ctl.Patch)

	// Soft delete + restore
	g.Delete("/:id", ctl.Delete)
	g.Post("/:id/restore", ctl.Restore)


	ctl2 := controller.NewClassRoomVirtualLinkController(db)
	h := admin.Group("/class-room-virtual-links")

	// Create / Read
	h.Post("/", ctl2.Create)
	h.Get("/", ctl2.List)
	h.Get("/:id", ctl2.Detail)

	// Update
	h.Patch("/:id", ctl2.Update)

	// Soft delete + restore (+ hard delete via ?hard=true)
	h.Delete("/:id", ctl2.Delete)
	h.Post("/:id/restore", ctl2.Restore)
}
