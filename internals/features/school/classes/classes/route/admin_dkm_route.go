// internals/route/classes_admin_routes.go
package route

import (
	classctrl "masjidku_backend/internals/features/school/classes/classes/controller"
	masjidkuMiddleware "masjidku_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)


 func ClassAdminRoutes(admin fiber.Router, db *gorm.DB) {
	classHandler := classctrl.NewClassController(db)

	// kalau ada middleware versi by-param, pakai itu:
	// grp := admin.Group("/:masjid_id/classes", masjidkuMiddleware.IsMasjidAdminByParam("masjid_id"))

	grp := admin.Group("/:masjid_id/classes", masjidkuMiddleware.IsMasjidAdmin())
	{
		grp.Post("/",          classHandler.CreateClass)
		grp.Patch("/:id",      classHandler.PatchClass)
		grp.Delete("/:id",     classHandler.SoftDeleteClass)
	}

	// Controller class parents
	parentHandler := classctrl.NewClassParentController(db, nil)

	// Prefix masjid_id biar ResolveMasjidContext dapat konteks langsung
	classParents := admin.Group("/:masjid_id/class-parents", masjidkuMiddleware.IsMasjidAdmin())
	{
		classParents.Post("/", parentHandler.Create)
		classParents.Patch("/:id", parentHandler.Patch)
		classParents.Delete("/:id", parentHandler.Delete)
	}
}
