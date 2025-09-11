// internals/server/routes/announcement_theme_routes.go
package routes

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	annCtl "masjidku_backend/internals/features/school/announcements/announcement_thema/controller"
)

// Dipanggil dari LembagaAdminRoutes(r, db)
// Asumsikan r SUDAH diproteksi middleware auth + isMasjidAdmin di level atas.
func AnnouncementAdminRoute(r fiber.Router, db *gorm.DB) {
	ctl := annCtl.NewAnnouncementThemeController(db)

	themes := r.Group("/announcement-themes")

	themes.Post("/", ctl.Create)      // Create
	themes.Get("/list", ctl.List)         // List
	themes.Put("/:id", ctl.Update)    // Update
	themes.Delete("/:id", ctl.Delete) // Soft delete
}
