// internals/route/classes_all_routes.go
package route

import (
	classParentctrl "madinahsalam_backend/internals/features/school/classes/class_parents/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AllClassParentRoutes(admin fiber.Router, db *gorm.DB) {
	h := classParentctrl.NewClassParentController(db, nil)

	classParents := admin.Group("/:school_id/class-parents")
	classParents.Get("/list", h.List)
}
