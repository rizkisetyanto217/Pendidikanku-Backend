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
	user_ctl := sppCtl.NewUserSppBillingItemController(db)

	spp := r.Group("/spp-billings")

	spp.Post("/", ctl.Create)     // POST   /spp/billings
	spp.Get("/", ctl.List)        // GET    /spp/billings
	spp.Get("/:id", ctl.GetByID) // GET    /spp/billings/:id
	spp.Put("/:id", ctl.Update) // PUT /spp/billings/:id
	spp.Delete("/:id", ctl.Delete) // DELETE /spp/billings/:id

	user_spp := r.Group("/user-spp-billings")

	user_spp.Get("/", user_ctl.List)
	user_spp.Get("/:id", user_ctl.GetByID)
	user_spp.Patch("/:id", user_ctl.Update)
	user_spp.Delete("/:id", user_ctl.Delete)
}
