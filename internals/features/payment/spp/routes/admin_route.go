// internals/routes/spp_billing_routes.go
package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	sppCtl "masjidku_backend/internals/features/payment/spp/controller"
)

// Contoh pemakaian: route.SppBillingRoutes(app, db)
func SppBillingAdminRoutes(r fiber.Router, db *gorm.DB) {
	ctl := sppCtl.NewSppBillingController(db)

	spp := r.Group("/spp-billings")

	spp.Post("/", ctl.Create)     // POST   /spp/billings
	spp.Get("/", ctl.List)        // GET    /spp/billings
	spp.Get("/:id", ctl.GetByID) // GET    /spp/billings/:id
	spp.Patch("/:id", ctl.Update) // PATCH /spp/billings/:id
	spp.Delete("/:id", ctl.Delete) // DELETE /spp/billings/:id
}
