package route

import (
	"masjidku_backend/internals/constants"
	homeController "masjidku_backend/internals/features/home/articles/controller"
	authMiddleware "masjidku_backend/internals/middlewares/auth"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// 🔐 Admin/DKM/Owner only (CRUD Article & Carousel)
func ArticleAdminRoutes(router fiber.Router, db *gorm.DB) {
	articleCtrl := homeController.NewArticleController(db)
	carouselCtrl := homeController.NewCarouselController(db)

	// Group besar: wajib login + role admin/dkm/owner
	adminOrOwner := router.Group("/",
		authMiddleware.AuthMiddleware(db),
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola artikel & carousel"),
			constants.AdminAndAbove, // admin, dkm, owner
		),
		// NOTE: Tidak pakai IsMasjidAdmin() karena konteksnya 'home/global'.
	)

	// === Article (CRUD)
	article := adminOrOwner.Group("/articles")
	article.Post("/", articleCtrl.CreateArticle)      // ➕ Buat artikel
	article.Put("/:id", articleCtrl.UpdateArticle)    // 🔄 Update artikel
	article.Delete("/:id", articleCtrl.DeleteArticle) // 🗑️ Hapus artikel

	// === Carousel (CRUD + list admin)
	carousel := adminOrOwner.Group("/carousels")
	carousel.Post("/", carouselCtrl.CreateCarousel)      // ➕ Buat carousel
	carousel.Put("/:id", carouselCtrl.UpdateCarousel)    // 🔄 Update carousel
	carousel.Delete("/:id", carouselCtrl.DeleteCarousel) // ❌ Hapus carousel
	carousel.Get("/", carouselCtrl.GetAllCarouselsAdmin) // 📄 List seluruh carousel (termasuk non-aktif)
}
