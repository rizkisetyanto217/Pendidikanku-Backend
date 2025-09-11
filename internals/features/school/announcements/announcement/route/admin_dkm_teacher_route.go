package routes

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	annCtl "masjidku_backend/internals/features/school/announcements/announcement/controller"
)

// Rute ADMIN/TEACHER untuk Announcements.
// Catatan: pastikan router 'r' sudah merupakan group /admin dan sudah ada middleware auth di level atas.
func AnnouncementAdminRoutes(r fiber.Router, db *gorm.DB) {
	ctl := annCtl.NewAnnouncementController(db)
	ctl2 := annCtl.NewAnnouncementURLController(db)


	grp := r.Group("/announcements") // hasil akhir: /admin/announcements


	grp.Get("/list", ctl.List)      // ← get all
	grp.Post("/", ctl.Create)      // Create (Admin: global; Teacher: wajib section)

	grp.Put("/:id", ctl.Update)    // Update (role-aware)
	grp.Delete("/:id", ctl.Delete) // Delete (role-aware)


	
	grp2 := r.Group("/announcement-urls")

	// list & create (tanpa/ dengan trailing slash)
	grp2.Get("/list", ctl2.List)
	grp2.Post("/", ctl2.Create)

	// detail & update & delete & restore
	// grp2.Get("/:id", ctl2.Detail)
	grp2.Patch("/:id", ctl2.Update)
	grp2.Delete("/:id", ctl2.Delete)
	// grp2.Patch("/:id/restore", ctl2.Restore)

	// (opsional, kalau sudah ada handlernya)
	// grp.Get("/", ctl2.List)
	// grp.Get("/:id", ctl2.GetByID)
}
