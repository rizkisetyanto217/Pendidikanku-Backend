// internals/route/classes_admin_routes.go
package route

import (
	classctrl "schoolku_backend/internals/features/school/classes/classes/controller"
	schoolkuMiddleware "schoolku_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func ClassAdminRoutes(admin fiber.Router, db *gorm.DB) {
	classHandler := classctrl.NewClassController(db)

	// kalau ada middleware versi by-param, pakai itu:
	// grp := admin.Group("/:school_id/classes", schoolkuMiddleware.IsSchoolAdminByParam("school_id"))

	grp := admin.Group("/:school_id/classes", schoolkuMiddleware.IsSchoolAdmin())
	{
		grp.Post("/", classHandler.CreateClass)
		grp.Patch("/:id", classHandler.PatchClass)
		grp.Delete("/:id", classHandler.SoftDeleteClass)
	}

	// Controller class parents
	parentHandler := classctrl.NewClassParentController(db, nil)

	// Prefix school_id biar ResolveSchoolContext dapat konteks langsung
	classParents := admin.Group("/:school_id/class-parents", schoolkuMiddleware.IsSchoolAdmin())
	{
		classParents.Post("/", parentHandler.Create)
		classParents.Patch("/:id", parentHandler.Patch)
		classParents.Delete("/:id", parentHandler.Delete)
	}

	// ================================
	// Student Class Enrollments
	// ================================
	enrollHandler := classctrl.NewStudentClassEnrollmentController(db)

	// kalau ada middleware versi by-param, bisa:
	// enrollGrp := admin.Group("/:school_id/class-enrollments", schoolkuMiddleware.IsSchoolAdminByParam("school_id"))

	enrollGrp := admin.Group("/:school_id/class-enrollments", schoolkuMiddleware.IsSchoolAdmin())
	{
		// LIST: GET /:school_id/class-enrollments
		enrollGrp.Get("/", enrollHandler.List)

		// (opsional, siapin slot kalau nanti ada)
		// enrollGrp.Post("/", enrollHandler.Create)
		// enrollGrp.Patch("/:id", enrollHandler.Patch)
		// enrollGrp.Delete("/:id", enrollHandler.Delete)
	}
}
