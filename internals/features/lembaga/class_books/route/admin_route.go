// internals/features/lembaga/library/route/library_admin_routes.go
package route

import (
	cbController "masjidku_backend/internals/features/lembaga/class_books/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Panggil ini dengan r = app.Group("/admin")
func ClassBooksAdminRoutes(r fiber.Router, db *gorm.DB) {
	// Controllers
	booksCtl := &cbController.BooksController{DB: db}
	csbCtl := &cbController.ClassSubjectBookController{DB: db}

	// -----------------------------
	// /admin/class-books
	// -----------------------------
	books := r.Group("/books")
	books.Post("/", booksCtl.Create)      // POST   /admin/class-books
	books.Get("/", booksCtl.List)         // GET    /admin/class-books
	books.Get("/:id", booksCtl.GetByID)   // GET    /admin/class-books/:id
	books.Put("/:id", booksCtl.Update)    // PUT    /admin/class-books/:id
	books.Delete("/:id", booksCtl.Delete) // DELETE /admin/class-books/:id (soft delete)

	// -----------------------------
	// /admin/class-subject-books (relasi class_subject <-> book)
	// -----------------------------
	csb := r.Group("/class-subject-books")
	csb.Post("/", csbCtl.Create)        // POST   /admin/class-subject-books
	csb.Get("/", csbCtl.List)           // GET    /admin/class-subject-books?class_subject_id=&book_id=&...
	csb.Get("/:id", csbCtl.GetByID)     // GET    /admin/class-subject-books/:id
	csb.Put("/:id", csbCtl.Update)      // PUT    /admin/class-subject-books/:id
	csb.Delete("/:id", csbCtl.Delete)   // DELETE /admin/class-subject-books/:id (soft/hard via ?force=true)
}
