// internals/route/ui_theme_admin_routes.go
package route

import (
	uictl "masjidku_backend/internals/features/lembaga/ui/theme/controller"
	masjidkuMiddleware "masjidku_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Khusus owner/admin: create, patch, delete (custom preset)
func UIThemeAdminRoutes(admin fiber.Router, db *gorm.DB) {
	h := uictl.NewUIThemeCustomPresetController(db, nil)

	// /admin/ui-theme-custom-presets
	r := admin.Group("/ui-theme-custom-presets", masjidkuMiddleware.IsMasjidAdmin())
	{
		r.Post("/", h.Create)      // create custom preset
		r.Patch("/:id", h.Patch)   // patch (partial + JSON merge)
		r.Delete("/:id", h.Delete) // delete
	}

		// ===== Theme Choices (per masjid) =====
	choice := uictl.NewUIThemeChoiceController(db, nil)
	chGroup := admin.Group("/ui-theme-choices", masjidkuMiddleware.IsMasjidAdmin())
	{
		chGroup.Post("/", choice.Create)      // create choice
		chGroup.Patch("/:id", choice.Patch)   // patch choice
		chGroup.Delete("/:id", choice.Delete) // delete choice
		// GET untuk choice tersedia di route public
	}
}
