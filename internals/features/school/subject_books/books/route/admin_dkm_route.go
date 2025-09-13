package route

import (
	"masjidku_backend/internals/constants"
	cbController "masjidku_backend/internals/features/school/subject_books/books/controller"
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
	books := r.Group("/books", adminGuard)
	books.Post("/",   booksCtl.Create)
	books.Put("/:id", booksCtl.Update)
	books.Delete("/:id", booksCtl.Delete)

	// /api/a/class-books/class-subject-books
	csb := r.Group("/class-subject-books", adminGuard)
	csb.Post("/",     csbCtl.Create)
	csb.Put("/:id",   csbCtl.Update)
	csb.Delete("/:id", csbCtl.Delete)

}
