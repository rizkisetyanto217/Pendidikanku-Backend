// file: internals/features/lembaga/classes/user_classes/main/route/user_routes.go
package route

import (
	// Classes controller (read only)
	classParentCtrl "madinahsalam_backend/internals/features/school/classes/class_parents/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func ClassParentUserRoutes(r fiber.Router, db *gorm.DB) {
	// ===== Controllers =====
	classParentHendler := classParentCtrl.NewClassParentController(db, nil)

	// ================================
	// Class Parents (READ-ONLY untuk user)
	// ================================
	// Mirror admin: /class-parents
	classParents := r.Group("/class-parents")
	{
		// GET /api/u/class-parents/list
		classParents.Get("/list", classParentHendler.List)
	}

}
