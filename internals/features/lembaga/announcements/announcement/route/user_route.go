package routes

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	annCtl "masjidku_backend/internals/features/lembaga/announcements/announcement/controller"
)

// Rute ADMIN/TEACHER untuk Announcements.
// Catatan: pastikan router 'r' sudah merupakan group /admin dan sudah ada middleware auth di level atas.
func AnnouncementUserRoutes(r fiber.Router, db *gorm.DB) {
	ctl := annCtl.NewAnnouncementController(db)

	grp := r.Group("/announcements") // hasil akhir: /admin/announcements


	grp.Get("/", ctl.List)      // ← get all
	grp.Get("/:id", ctl.GetByID) // ← get by id

	grp.Post("/", ctl.Create)      // Create (Admin: global; Teacher: wajib section)
	grp.Put("/:id", ctl.Update)    // Update (role-aware)
	grp.Delete("/:id", ctl.Delete) // Delete (role-aware)

	// (opsional, kalau sudah ada handlernya)
	// grp.Get("/", ctl.List)
	// grp.Get("/:id", ctl.GetByID)
}
