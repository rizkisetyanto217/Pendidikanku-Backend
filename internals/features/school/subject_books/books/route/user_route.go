package route

import (
	cbController "masjidku_backend/internals/features/school/subject_books/books/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Panggil dengan: route.ClassBooksUserRoutes(app.Group("/api/u/class-books"), db)
// Hasil endpoint:
//   /api/u/class-books
//   /api/u/class-subject-books
func ClassBooksUserRoutes(r fiber.Router, db *gorm.DB) {
	booksCtl := &cbController.BooksController{DB: db}
	csbCtl   := &cbController.ClassSubjectBookController{DB: db}
	bookURLCtl := cbController.NewBookURLController(db) // ⬅️ tambah ini

	// /api/u/class-books
	books := r.Group("/books")
	books.Get("/list", booksCtl.List)

	// /api/u/class-books/class-subject-books
	csb := r.Group("/class-subject-books")
	csb.Get("/list", csbCtl.List)
	csb.Get("/:id", csbCtl.GetByID)

	// /api/u/class-books/book-urls  ⬅️ TAMBAHAN (read-only)
	urls := r.Group("/book-urls")
	urls.Get("/filter", bookURLCtl.Filter)
	urls.Get("/:id",    bookURLCtl.GetByID)
}
