package routes

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	annCtl "masjidku_backend/internals/features/school/others/announcements/controller"
)

// Rute ADMIN/TEACHER (harus sudah di-mount di /admin dan ada middleware auth di atasnya)
func AnnouncementAdminRoutes(r fiber.Router, db *gorm.DB) {
	ann := annCtl.NewAnnouncementController(db)
	theme := annCtl.NewAnnouncementThemeController(db)

	// ================== ANNOUNCEMENTS ==================

	// /admin/:masjid_id/announcements
	withID := r.Group("/:masjid_id/announcements")
	withID.Post("/", ann.Create)
	withID.Put("/:id", ann.Update)
	withID.Delete("/:id", ann.Delete)
	withID.Get("/list", ann.List)

	// /admin/by-slug/:masjid_slug/announcements
	withSlug := r.Group("/by-slug/:masjid_slug/announcements")
	withSlug.Post("/", ann.Create)
	withSlug.Put("/:id", ann.Update)
	withSlug.Delete("/:id", ann.Delete)
	withSlug.Get("/list", ann.List)

	// ================== ANNOUNCEMENT THEMES ==================

	// /admin/:masjid_id/announcement-themes
	themesID := r.Group("/:masjid_id/announcement-themes")
	themesID.Post("/", theme.Create)
	themesID.Put("/:id", theme.Update)
	themesID.Delete("/:id", theme.Delete)
	// (opsional) kalau nanti ada List: themesID.Get("/list", theme.List)

	// /admin/by-slug/:masjid_slug/announcement-themes
	themesSlug := r.Group("/by-slug/:masjid_slug/announcement-themes")
	themesSlug.Post("/", theme.Create)
	themesSlug.Put("/:id", theme.Update)
	themesSlug.Delete("/:id", theme.Delete)
	// (opsional) kalau nanti ada List: themesSlug.Get("/list", theme.List)
}
