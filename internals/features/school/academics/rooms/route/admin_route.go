// file: internals/features/school/schedule_rooms/rooms/routes/class_room_admin_routes.go
package routes

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	clsCtl "schoolku_backend/internals/features/school/academics/rooms/controller"
)

// RoomsAdminRoutes — route khusus ADMIN (CRUD penuh + restore).
// Contoh mount dari caller:
//
//	admin := api.Group("/api/a") // atau sesuai prefix kamu
//	routes.RoomsAdminRoutes(admin, db)
func RoomsAdminRoutes(admin fiber.Router, db *gorm.DB) {
	ctl := clsCtl.NewClassRoomController(db, nil) // validator nil
	// Tambah :school_id biar ResolveSchoolContext bisa resolve dari path
	g := admin.Group("/:school_id/class-rooms")

	// Write
	g.Post("/", ctl.Create)
	g.Patch("/:id", ctl.Patch)

	// Soft delete + restore
	g.Delete("/:id", ctl.Delete)
	g.Post("/:id/restore", ctl.Restore)

}
