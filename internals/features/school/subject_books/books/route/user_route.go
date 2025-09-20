package route

import (
	cbController "masjidku_backend/internals/features/school/subject_books/books/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Panggil dengan: route.ClassBooksUserRoutes(app.Group("/api/u"), db)
// Hasil endpoint:
//   GET /api/u/:masjid_id/books/list
//   GET /api/u/:masjid_id/class-subject-books/list
func ClassBooksUserRoutes(r fiber.Router, db *gorm.DB) {
	booksCtl := &cbController.BooksController{DB: db}
	csbCtl := &cbController.ClassSubjectBookController{DB: db}

	// base group pakai masjid_id di path
	g := r.Group("/:masjid_id")

	// /api/u/:masjid_id/books/list
	books := g.Group("/books")
	books.Get("/list", booksCtl.List)

	// /api/u/:masjid_id/class-subject-books/list
	csb := g.Group("/class-subject-books")
	csb.Get("/list", csbCtl.List)
}
