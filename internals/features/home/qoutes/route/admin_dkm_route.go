package route

import (
	"masjidku_backend/internals/constants"
	homeController "masjidku_backend/internals/features/home/qoutes/controller"
	authMiddleware "masjidku_backend/internals/middlewares/auth"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// 🔐 Admin/DKM/Owner only
func QuoteAdminRoutes(router fiber.Router, db *gorm.DB) {
	ctrl := homeController.NewQuoteController(db)

	admin := router.Group("/quotes",
		authMiddleware.AuthMiddleware(db),
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola quotes"),
			constants.AdminAndAbove, // admin, dkm, owner
		),
	)

	admin.Post("/", ctrl.CreateQuote)           // ➕ Buat quote
	admin.Post("/batch", ctrl.CreateQuotes)     // ➕ Tambah banyak quote
	admin.Put("/:id", ctrl.UpdateQuote)         // ✏️ Ubah quote
	admin.Delete("/:id", ctrl.DeleteQuote)      // 🗑️ Hapus quote
	admin.Get("/", ctrl.GetAllQuotes)           // 📄 Lihat semua (termasuk non-publish kalau controllernya dukung)
	admin.Get("/:id", ctrl.GetQuoteByID)        // 🔍 Detail quote
}
