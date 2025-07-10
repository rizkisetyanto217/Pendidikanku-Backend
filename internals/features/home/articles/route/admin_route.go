package route

import (
	"masjidku_backend/internals/features/home/articles/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func ArticleAdminRoutes(api fiber.Router, db *gorm.DB) {
	articleCtrl := controller.NewArticleController(db)

	// === ADMIN ROUTES ===
	article := api.Group("/articles")
	article.Post("/", articleCtrl.CreateArticle)      // ➕ Buat artikel baru
	article.Put("/:id", articleCtrl.UpdateArticle)    // 🔄 Perbarui artikel
	article.Delete("/:id", articleCtrl.DeleteArticle) // 🗑️ Hapus artikel

	carouselCtrl := controller.NewCarouselController(db)
	// === ADMIN ROUTES ===
	carousel := api.Group("/carousels")
	carousel.Post("/", carouselCtrl.CreateCarousel)      // ➕ Buat carousel
	carousel.Put("/:id", carouselCtrl.UpdateCarousel)    // 🔄 Update carousel
	carousel.Delete("/:id", carouselCtrl.DeleteCarousel) // ❌ Hapus carousel
	carousel.Get("/", carouselCtrl.GetAllCarouselsAdmin) // 📄 Lihat semua carousel (termasuk non-aktif)
}
