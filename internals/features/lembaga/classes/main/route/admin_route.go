// internals/route/classes_admin_routes.go
package route

import (
	classctrl "masjidku_backend/internals/features/lembaga/classes/main/controller"
	masjidkuMiddleware "masjidku_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func ClassAdminRoutes(admin fiber.Router, db *gorm.DB) {
	h := classctrl.NewClassController(db)

	// /admin/classes (semua pakai IsMasjidAdmin)
	classes := admin.Group("/classes", masjidkuMiddleware.IsMasjidAdmin())

	classes.Post("/", h.CreateClass)
	classes.Get("/", h.ListClasses)
	classes.Get("/slug/:slug", h.GetClassBySlug)
	classes.Get("/:id", h.GetClassByID)
	classes.Put("/:id", h.UpdateClass)
	classes.Delete("/:id", h.SoftDeleteClass)
}
