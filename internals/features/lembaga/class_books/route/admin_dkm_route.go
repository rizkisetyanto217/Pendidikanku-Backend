package route

import (
	"masjidku_backend/internals/constants"
	cbController "masjidku_backend/internals/features/lembaga/class_books/controller"
	"masjidku_backend/internals/middlewares/auth"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Panggil dengan: route.ClassBooksAdminRoutes(app.Group("/api/a/class-books"), db)
// Hasil endpoint:
//   /api/a/class-books
//   /api/a/class-subject-books
func ClassBooksAdminRoutes(r fiber.Router, db *gorm.DB) {
	booksCtl := &cbController.BooksController{DB: db}
	csbCtl   := &cbController.ClassSubjectBookController{DB: db}

	// Wajib role admin/dkm/owner
	adminGuard := auth.OnlyRolesSlice(
		constants.RoleErrorAdmin("aksi ini untuk admin/owner"),
		constants.AdminAndAbove,
	)

	// /api/a/class-books
	books := r.Group("/", adminGuard)
	books.Post("/",   booksCtl.Create)            // ➕ buat buku
	books.Get("/",    booksCtl.ListWithUsages)    // 📄 list + usages
	books.Get("/:id", booksCtl.GetWithUsagesByID) // 📄 detail
	books.Put("/:id", booksCtl.Update)            // ✏️ update
	books.Delete("/:id", booksCtl.Delete)         // 🗑️ soft delete

	// /api/a/class-subject-books
	csb := r.Group("/class-subject-books", adminGuard)
	csb.Post("/",     csbCtl.Create)   // ➕ pasang buku ke subject
	csb.Get("/",      csbCtl.List)     // 📄 list relasi
	csb.Get("/:id",   csbCtl.GetByID)  // 📄 detail relasi
	csb.Put("/:id",   csbCtl.Update)   // ✏️ update relasi
	csb.Delete("/:id", csbCtl.Delete)  // 🗑️ soft/hard
}
