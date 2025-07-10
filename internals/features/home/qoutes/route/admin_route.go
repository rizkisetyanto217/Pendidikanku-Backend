package route

import (
	"masjidku_backend/internals/features/home/qoutes/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func QuoteAdminRoutes(api fiber.Router, db *gorm.DB) {
	ctrl := controller.NewQuoteController(db)

	// === ADMIN ROUTES ===
	admin := api.Group("/quotes")
	admin.Post("/", ctrl.CreateQuote)              // ➕ Buat quote
	admin.Post("/batch", ctrl.CreateQuotes) // ➕ Tambah banyak quote
	admin.Put("/:id", ctrl.UpdateQuote)            // ✏️ Ubah quote
	admin.Delete("/:id", ctrl.DeleteQuote)         // 🗑️ Hapus quote
	admin.Get("/", ctrl.GetAllQuotes)              // 📄 Lihat semua quote
	admin.Get("/:id", ctrl.GetQuoteByID)           // 🔍 Detail quote
}
