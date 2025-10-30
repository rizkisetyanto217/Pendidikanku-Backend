// internals/route/classes_all_routes.go
package route

import (
	classctrl "masjidku_backend/internals/features/school/classes/classes/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AllClassRoutes(admin fiber.Router, db *gorm.DB) {
	h := classctrl.NewClassController(db)

	// /admin/:masjid_id/classes (READ endpoints umum)
	classes := admin.Group("/:masjid_id/classes")
	classes.Get("/list", h.ListClasses)
	classes.Get("/slug/:slug", h.GetClassBySlug)

	// /admin/:masjid_id/class-parents (READ endpoints umum)
	cp := classctrl.NewClassParentController(db, nil)
	classParents := admin.Group("/:masjid_id/class-parents")
	classParents.Get("/list", cp.List)
}
