// internals/route/classes_all_routes.go
package route

import (
	classctrl "madinahsalam_backend/internals/features/school/classes/classes/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AllClassRoutes(admin fiber.Router, db *gorm.DB) {
	h := classctrl.NewClassController(db)

	// /admin/:school_id/classes (READ endpoints umum)
	classes := admin.Group("/:school_id/classes")
	classes.Get("/list", h.ListClasses)

}
