// internals/route/classes_all_routes.go
package route

import (
	classctrl "masjidku_backend/internals/features/school/classes/classes/controller"


	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func ClassAllRoutes(admin fiber.Router, db *gorm.DB) {
	h := classctrl.NewClassController(db)

	// /admin/classes (READ endpoints umum)
	classes := admin.Group("/classes")
	classes.Get("/list", h.ListClasses)
	classes.Get("/slug/:slug", h.GetClassBySlug)

	// /admin/class-parents (READ endpoints umum)
	cp := classctrl.NewClassParentController(db, nil)
	classParents := admin.Group("/class-parents")
	classParents.Get("/list", cp.List)
}
