// file: internals/features/academics/books/route/class_books_user_route.go
package route

import (
	bookController "madinahsalam_backend/internals/features/school/academics/books/controller/books"
	ClassSubjectBooksController "madinahsalam_backend/internals/features/school/academics/books/controller/class_subject_books"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Panggil dengan: AllClassBooksRoutes(app.Group("/api/u"), db)
//
// Hasil endpoint:
//
//   - GET /api/u/i/:school_id/books/list
//   - GET /api/u/i/:school_id/class-subject-books/list
//   - GET /api/u/s/:school_slug/books/list
//   - GET /api/u/s/:school_slug/class-subject-books/list
//
// Resolver school di controller sudah token-aware + slug-aware:
//  1. Coba active_school_id dari token,
//  2. fallback ke ResolveSchoolContext (baca :school_id atau :school_slug).
func AllClassBooksRoutes(r fiber.Router, db *gorm.DB) {
	booksCtl := &bookController.BooksController{DB: db}
	csbCtl := &ClassSubjectBooksController.ClassSubjectBookController{DB: db}

	// =========================
	// 1) By school_id (UUID)
	//    /api/u/i/:school_id/...
	// =========================
	rByID := r.Group("/i/:school_id")

	// /api/u/i/:school_id/books/list
	booksByID := rByID.Group("/books")
	booksByID.Get("/list", booksCtl.List)

	// /api/u/i/:school_id/class-subject-books/list
	csbByID := rByID.Group("/class-subject-books")
	csbByID.Get("/list", csbCtl.List)

	// =========================
	// 2) By school_slug
	//    /api/u/s/:school_slug/...
	// =========================
	rBySlug := r.Group("/s/:school_slug")

	// /api/u/s/:school_slug/books/list
	booksBySlug := rBySlug.Group("/books")
	booksBySlug.Get("/list", booksCtl.List)

	// /api/u/s/:school_slug/class-subject-books/list
	csbBySlug := rBySlug.Group("/class-subject-books")
	csbBySlug.Get("/list", csbCtl.List)
}
