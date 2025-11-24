// internals/route/ui_theme_owner_routes.go
package route

import (
	uictl "madinahsalam_backend/internals/features/lembaga/ui/theme/controller"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Khusus owner: create, patch, delete
func UIThemeOwnerRoutes(owner fiber.Router, db *gorm.DB) {
	h := uictl.NewUIThemePresetController(db, nil)

	// /owner/ui-theme-presets  (proteksi: Owner)
	r := owner.Group("/ui-theme-presets", helperAuth.OwnerOnly())
	{
		r.Post("/", h.Create)      // create preset
		r.Patch("/:id", h.Patch)   // patch (partial + JSON merge)
		r.Delete("/:id", h.Delete) // soft delete
	}
}
