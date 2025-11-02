package route

import (
	homeController "schoolku_backend/internals/features/home/qoutes/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// 🌐 All/Public (read-only)
func AllQuoteRoutes(router fiber.Router, db *gorm.DB) {
	ctrl := homeController.NewQuoteController(db)

	public := router.Group("/quotes")
	public.Get("/", ctrl.GetAllQuotes)             // 📄 Semua quote (umumnya yang aktif/published)
	public.Get("/:id", ctrl.GetQuoteByID)          // 🔍 Detail quote
	public.Get("/batch/30", ctrl.GetQuotesByBatch) // 📦 Ambil 30 quote per batch (sesuai controller)
}
