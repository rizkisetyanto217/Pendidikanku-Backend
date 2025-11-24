package routes

// import (
// 	"github.com/gofiber/fiber/v2"
// 	"gorm.io/gorm"

// 	annCtl "madinahsalam_backend/internals/features/school/others/post/controller"
// )

// // Rute USER (bisa juga dipakai Admin/Teacher tergantung middleware di atas 'r')
// func PostUserRoutes(r fiber.Router, db *gorm.DB) {
// 	ann := annCtl.NewPostController(db)
// 	thr := annCtl.NewPostThemeController(db, nil)

// 	// =============== SCOPED BY MASJID UUID ====================
// 	// Contoh: /:school_id/announcements
// 	withID := r.Group("/:school_id")
// 	// Announcements
// 	annID := withID.Group("/announcements")
// 	annID.Post("/", ann.Create)
// 	annID.Get("/list", ann.List)

// 	// Announcement Themes (LIST & DETAIL)
// 	// Contoh: GET /:school_id/announcement-themes?include=announcements
// 	//         GET /:school_id/announcement-themes/:id?include=announcements
// 	thID := withID.Group("/announcement-themes")
// 	thID.Get("/", thr.List)    // list
// 	thID.Get("/:id", thr.List) // detail (List handler udah handle params("id"))

// 	// =============== SCOPED BY MASJID SLUG ====================
// 	// Contoh: /by-slug/:school_slug/announcements
// 	withSlug := r.Group("/by-slug/:school_slug")
// 	// Announcements
// 	annSlug := withSlug.Group("/announcements")
// 	annSlug.Post("/", ann.Create)
// 	annSlug.Get("/list", ann.List)

// 	// Announcement Themes (LIST & DETAIL)
// 	// Contoh: GET /by-slug/:school_slug/announcement-themes?include=announcements
// 	//         GET /by-slug/:school_slug/announcement-themes/:id?include=announcements
// 	thSlug := withSlug.Group("/announcement-themes")
// 	thSlug.Get("/", thr.List)    // list
// 	thSlug.Get("/:id", thr.List) // detail
// }
