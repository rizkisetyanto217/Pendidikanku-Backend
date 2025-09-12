package routes

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	annCtl "masjidku_backend/internals/features/school/announcements/announcement/controller"
)

// Rute ADMIN/TEACHER untuk Announcements.
// Catatan: pastikan router 'r' sudah merupakan group /admin dan sudah ada middleware auth di level atas.
func AnnouncementUserRoutes(r fiber.Router, db *gorm.DB) {
	ctl := annCtl.NewAnnouncementController(db)
	ctl2 := annCtl.NewAnnouncementURLController(db)

	grp := r.Group("/announcements") // hasil akhir: /admin/announcements
	grp.Post("/", ctl.Create)      // Create (Admin: global; Teacher: wajib section)
	grp.Get("/list", ctl.List)      // ← get all

	// (opsional, kalau sudah ada handlernya)
	// grp.Get("/", ctl.List)
	// grp.Get("/:id", ctl.GetByID)

	grp2 := r.Group("/announcement-urls")

	// list & detail saja untuk user
	grp2.Post("/", ctl2.Create)
	grp2.Get("/list", ctl2.List)
}
