// internals/routes/public_yayasan_routes.go
package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	yCtrl "masjidku_backend/internals/features/lembaga/yayasans/controller"
)

func AllYayasanRoutes(r fiber.Router, db *gorm.DB) {
	h := yCtrl.NewYayasanController(db)

	g := r.Group("/yayasans")
	g.Get("/", h.List)    // List public
	g.Get("/:id", h.Detail) // Detail by ID
}
