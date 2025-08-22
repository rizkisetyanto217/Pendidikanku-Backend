package route

import (
	homeController "masjidku_backend/internals/features/home/articles/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// 🌐 All/User (read-only)
func AllArticleRoutes(router fiber.Router, db *gorm.DB) {
	articleCtrl := homeController.NewArticleController(db)
	carouselCtrl := homeController.NewCarouselController(db)

	// === Article (read-only)
	article := router.Group("/articles")
	article.Get("/", articleCtrl.GetAllArticles)    // 📄 Semua artikel aktif/publik
	article.Get("/:id", articleCtrl.GetArticleByID) // 🔍 Detail artikel

	// === Carousel (read-only aktif saja)
	carousel := router.Group("/carousels")
	carousel.Get("/", carouselCtrl.GetAllActiveCarousels) // 🎡 Carousel aktif
}
