// internals/route/classes_all_routes.go
package route

import (
	classctrl "schoolku_backend/internals/features/school/classes/classes/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AllClassRoutes(admin fiber.Router, db *gorm.DB) {
	h := classctrl.NewClassController(db)

	// /admin/:school_id/classes (READ endpoints umum)
	classes := admin.Group("/:school_id/classes")
	classes.Get("/list", h.ListClasses)
	classes.Get("/slug/:slug", h.GetClassBySlug)

	// /admin/:school_id/class-parents (READ endpoints umum)
	cp := classctrl.NewClassParentController(db, nil)
	classParents := admin.Group("/:school_id/class-parents")
	classParents.Get("/list", cp.List)
}
