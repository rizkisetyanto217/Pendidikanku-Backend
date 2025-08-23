// internals/routes/spp_billing_routes.go
package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	sppCtl "masjidku_backend/internals/features/payment/spp/controller"
)

// Contoh pemakaian: route.SppBillingRoutes(app, db)
func SppBillingUserRoutes(r fiber.Router, db *gorm.DB) {
	user_ctl := sppCtl.NewUserSppBillingItemController(db)

	user_spp := r.Group("/user-spp-billings")

	user_spp.Get("/", user_ctl.List)
	user_spp.Get("/me", user_ctl.ListMine)
	user_spp.Get("/:id", user_ctl.GetByID)
	user_spp.Put("/:id", user_ctl.Update)
	user_spp.Delete("/:id", user_ctl.Delete)
}
