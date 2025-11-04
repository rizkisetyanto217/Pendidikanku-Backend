package routes

// import (
// 	"github.com/gofiber/fiber/v2"
// 	"gorm.io/gorm"

// 	annCtl "schoolku_backend/internals/features/school/others/post/controller"
// )

// // Rute ADMIN/TEACHER (harus sudah di-mount di /admin dan ada middleware auth di atasnya)
// func PostAdminRoutes(r fiber.Router, db *gorm.DB) {
// 	ann := annCtl.NewPostController(db)
// 	theme := annCtl.NewPostThemeController(db, nil)

// 	// ================== ANNOUNCEMENTS ==================

// 	// /admin/:school_id/announcements
// 	withID := r.Group("/:school_id/announcements")
// 	withID.Post("/", ann.Create)
// 	withID.Put("/:id", ann.Patch)
// 	withID.Delete("/:id", ann.Delete)
// 	withID.Get("/list", ann.List)

// 	// /admin/by-slug/:school_slug/announcements
// 	withSlug := r.Group("/by-slug/:school_slug/announcements")
// 	withSlug.Post("/", ann.Create)
// 	withSlug.Put("/:id", ann.Patch)
// 	withSlug.Delete("/:id", ann.Delete)
// 	withSlug.Get("/list", ann.List)

// 	// ================== ANNOUNCEMENT THEMES ==================

// 	// /admin/:school_id/announcement-themes
// 	themesID := r.Group("/:school_id/announcement-themes")
// 	themesID.Post("/", theme.Create)
// 	themesID.Put("/:id", theme.Patch)
// 	themesID.Delete("/:id", theme.Delete)
// 	// (opsional) kalau nanti ada List: themesID.Get("/list", theme.List)

// 	// /admin/by-slug/:school_slug/announcement-themes
// 	themesSlug := r.Group("/by-slug/:school_slug/announcement-themes")
// 	themesSlug.Post("/", theme.Create)
// 	themesSlug.Put("/:id", theme.Patch)
// 	themesSlug.Delete("/:id", theme.Delete)
// 	// (opsional) kalau nanti ada List: themesSlug.Get("/list", theme.List)
// }
