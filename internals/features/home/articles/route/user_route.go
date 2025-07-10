package route

import (
	"masjidku_backend/internals/features/home/articles/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AllArticleRoutes(api fiber.Router, db *gorm.DB) {
	articleCtrl := controller.NewArticleController(db)

	// === USER ROUTES ===
	article := api.Group("/articles")
	article.Get("/", articleCtrl.GetAllArticles)    // 📄 Lihat semua artikel
	article.Get("/:id", articleCtrl.GetArticleByID) // 🔍 Lihat detail artikel

	carouselCtrl := controller.NewCarouselController(db)
	// === USER ROUTES ===
	carousel := api.Group("/carousels")
	carousel.Get("/", carouselCtrl.GetAllActiveCarousels) // 🎡 Ambil semua carousel aktif
}
