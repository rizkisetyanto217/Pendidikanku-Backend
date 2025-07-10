package route

import (
	"masjidku_backend/internals/features/home/qoutes/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// func QuoteUserRoutes(api fiber.Router, db *gorm.DB) {
// 	ctrl := controller.NewQuoteController(db)

// 	// === USER ROUTES ===
// 	user := api.Group("/quotes")
// 	user.Get("/", ctrl.GetAllQuotes)             // 📄 Lihat semua quote
// 	user.Get("/:id", ctrl.GetQuoteByID)          // 🔍 Detail quote
// 	user.Get("/batch/30", ctrl.GetQuotesByBatch) // 📦 Ambil 30 quote per batch (gunakan query param ?batch_number=1)

// }

func AllQuoteRoutes(api fiber.Router, db *gorm.DB) {
	ctrl := controller.NewQuoteController(db)

	api.Get("/quotes", ctrl.GetAllQuotes)              // 📄 Lihat semua quote
	api.Get("/quotes/:id", ctrl.GetQuoteByID)          // 🔍 Detail quote
	api.Get("/quotes/batch/30", ctrl.GetQuotesByBatch) // 📦 Ambil 30 quote per batch
}
