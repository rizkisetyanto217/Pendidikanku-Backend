// file: internals/features/school/schedule_rooms/rooms/routes/class_room_user_routes.go
package routes

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	clsCtl "madinahsalam_backend/internals/features/school/academics/rooms/controller"
	schoolkuMiddleware "madinahsalam_backend/internals/middlewares/features"
)

// RoomsUserRoutes — route USER (read-only).
// Contoh mount dari caller:
//
//	user := api.Group("/api/u") // atau "/api/p" jika publik
//	routes.AllRoomsRoutes(user, db)
//
// Sehingga endpoint jadi:
//   - GET /api/u/:school_id/class-rooms/list
//   - GET /api/u/:school_slug/class-rooms/list
//   - (atau /api/p/... kalau dipasang di grup publik)
func AllRoomsRoutes(user fiber.Router, db *gorm.DB) {
	ctl := clsCtl.NewClassRoomController(db, nil)

	// ===== Base by school_id =====
	gByID := user.Group("/:school_id/class-rooms")
	gByID.Get("/list", ctl.List)

	// ===== Base by school_slug =====
	// UseSchoolScope biasanya ngasih tahu resolver bahwa param ini slug
	gBySlug := user.Group("/:school_slug/class-rooms",
		schoolkuMiddleware.UseSchoolScope(),
	)
	gBySlug.Get("/list", ctl.List)
}
