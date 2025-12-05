// internals/route/classes_admin_routes.go
package route

import (
	classParentctrl "madinahsalam_backend/internals/features/school/classes/class_parents/controller"

	schoolkuMiddleware "madinahsalam_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func ClassParentAdminRoutes(admin fiber.Router, db *gorm.DB) {

	// Controller class parents
	parentHandler := classParentctrl.NewClassParentController(db, nil)

	// Prefix school_id biar ResolveSchoolContext dapat konteks langsung
	classParents := admin.Group("/class-parents", schoolkuMiddleware.IsSchoolAdmin())
	{
		classParents.Post("/", parentHandler.Create)
		classParents.Patch("/:id", parentHandler.Patch)
		classParents.Delete("/:id", parentHandler.Delete)
	}
}
