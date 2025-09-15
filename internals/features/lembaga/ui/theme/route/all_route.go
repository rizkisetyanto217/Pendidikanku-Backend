package route

import (
	uictl "masjidku_backend/internals/features/lembaga/ui/theme/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Hanya GET (public)
func UIThemePublicRoutes(public fiber.Router, db *gorm.DB) {
	// --- Master Presets (sistem) ---
	hPreset := uictl.NewUIThemePresetController(db, nil)
	p := public.Group("/ui-theme-presets")
	{
		// GET gabungan (list / get by id via query ?id=UUID)
		p.Get("/list", hPreset.Get)
	}

	// --- Custom Presets (per masjid) ---
	hCustom := uictl.NewUIThemeCustomPresetController(db, nil)
	c := public.Group("/ui-theme-custom-presets")
	{
		// GET gabungan (list / get by id via query ?id=UUID, ?masjid_id=UUID)
		c.Get("/list", hCustom.Get)
	}

	// --- Theme Choices (per masjid) ---
	hChoice := uictl.NewUIThemeChoiceController(db, nil)
	ch := public.Group("/ui-theme-choices")
	{
		// GET gabungan (list / get by id via query ?id=UUID + filter: masjid_id, preset_id, custom_preset_id, is_default, is_enabled)
		ch.Get("/list", hChoice.Get)
	}
}
