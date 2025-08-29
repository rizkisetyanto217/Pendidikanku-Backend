package route

import (
	cbController "masjidku_backend/internals/features/school/class_subject_books/books/controller"

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

	// /api/u/class-books (read only)
	books := r.Group("/")
	books.Get("/",    booksCtl.ListWithUsages)
	books.Get("/:id", booksCtl.GetWithUsagesByID)

	// /api/u/class-subject-books (read only)
	csb := r.Group("/class-subject-books")
	csb.Get("/",    csbCtl.List)
	csb.Get("/:id", csbCtl.GetByID)
}
