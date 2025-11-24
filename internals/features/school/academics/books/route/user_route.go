package route

import (
	cbController "madinahsalam_backend/internals/features/school/academics/books/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Panggil dengan: route.ClassBooksUserRoutes(app.Group("/api/u"), db)
// Hasil endpoint:
//
//	GET /api/u/:school_id/books/list
//	GET /api/u/:school_id/class-subject-books/list
func ClassBooksUserRoutes(r fiber.Router, db *gorm.DB) {
	booksCtl := &cbController.BooksController{DB: db}
	csbCtl := &cbController.ClassSubjectBookController{DB: db}

	// /api/u/:school_id/books/list
	books := r.Group("/books")
	books.Get("/list", booksCtl.List)

	// /api/u/:school_id/class-subject-books/list
	csb := r.Group("/class-subject-books")
	csb.Get("/list", csbCtl.List)
}
