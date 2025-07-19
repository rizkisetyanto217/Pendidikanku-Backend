package details

import (
	DonationRoutes "masjidku_backend/internals/features/donations/donations/routes"
	AdviceRoutes "masjidku_backend/internals/features/home/advices/route"
	ArticleRoutes "masjidku_backend/internals/features/home/articles/route"
	FaqRoutes "masjidku_backend/internals/features/home/faqs/route"
	NotificationRoutes "masjidku_backend/internals/features/home/notifications/route"
	PostRoutes "masjidku_backend/internals/features/home/posts/route"
	QouteRoutes "masjidku_backend/internals/features/home/qoutes/route"
	QuestionnaireRoutes "masjidku_backend/internals/features/home/questionnaires/route"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// ✅ Untuk route publik tanpa token
// Contoh akses: /public/quotes
func HomePublicRoutes(api fiber.Router, db *gorm.DB) {
	QouteRoutes.AllQuoteRoutes(api, db)
	FaqRoutes.AllFaqQuestionRoutes(api, db)
	ArticleRoutes.AllArticleRoutes(api, db)
	PostRoutes.AllPostRoutes(api, db)
	QuestionnaireRoutes.AllQuestionnaireQuestionRoutes(api, db)
	NotificationRoutes.AllNotificationRoutes(api, db)
	DonationRoutes.DonationRoutes(api, db)
}

// ✅ Untuk route user login (dengan token)
// Contoh akses: /api/u/notifications
func HomePrivateRoutes(api fiber.Router, db *gorm.DB) {
	AdviceRoutes.AllAdviceRoutes(api, db)
}

// ✅ Untuk route admin masjid (token + admin)
// Contoh akses: /api/a/quotes
func HomeAdminRoutes(api fiber.Router, db *gorm.DB) {
	FaqRoutes.FaqQuestionAdminRoutes(api, db)
	AdviceRoutes.AdviceAdminRoutes(api, db)
	ArticleRoutes.ArticleAdminRoutes(api, db)
	PostRoutes.PostAdminRoutes(api, db)
	QuestionnaireRoutes.QuestionnaireQuestionAdminRoutes(api, db)
	QouteRoutes.QuoteAdminRoutes(api, db)
}
