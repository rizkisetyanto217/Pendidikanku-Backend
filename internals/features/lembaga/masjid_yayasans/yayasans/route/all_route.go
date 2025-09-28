// internals/routes/public_yayasan_routes.go
package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	yCtrl "masjidku_backend/internals/features/lembaga/masjid_yayasans/yayasans/controller"
)

func AllYayasanRoutes(r fiber.Router, db *gorm.DB) {
	h := yCtrl.NewYayasanController(db, nil, nil)

	g := r.Group("/yayasans")
	g.Get("/", h.List)    // List public
}
