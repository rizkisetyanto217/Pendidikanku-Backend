package route

import (
	"masjidku_backend/internals/constants"
	cbController "masjidku_backend/internals/features/school/academics/books/controller"
	"masjidku_backend/internals/middlewares/auth"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Panggil: route.ClassBooksAdminRoutes(app.Group("/api/a"), db)
// Endpoint hasil:
//
//	/api/a/:masjid_id/books
//	/api/a/:masjid_id/class-subject-books
//	/api/a/:masjid_id/book-urls
func ClassBooksAdminRoutes(r fiber.Router, db *gorm.DB) {
	booksCtl := &cbController.BooksController{DB: db}
	csbCtl := &cbController.ClassSubjectBookController{DB: db}

	// Book URL controller (punya Create, Patch, Delete)
	bookURLCtl := &cbController.BookURLController{
		DB:        db,
		Validator: validator.New(),
	}

	adminGuard := auth.OnlyRolesSlice(
		constants.RoleErrorAdmin("aksi ini untuk admin/owner"),
		constants.AdminAndAbove,
	)

	// ► Param pakai dash: :masjid_id
	g := r.Group("/:masjid_id", adminGuard)

	// Books
	books := g.Group("/books")
	books.Post("/", booksCtl.Create)
	books.Patch("/:id", booksCtl.Patch)
	books.Delete("/:id", booksCtl.Delete)

	// Class-Subject-Books
	csb := g.Group("/class-subject-books")
	csb.Post("/", csbCtl.Create)
	csb.Patch("/:id", csbCtl.Update)
	csb.Delete("/:id", csbCtl.Delete)

	// Book URLs
	bu := g.Group("/book-urls")
	bu.Post("/", bookURLCtl.Create)    // body berisi book_id, kind, dst
	bu.Patch("/:id", bookURLCtl.Patch) // partial update + handle primary & rotation object_key
	bu.Delete("/:id", bookURLCtl.Delete)
}
