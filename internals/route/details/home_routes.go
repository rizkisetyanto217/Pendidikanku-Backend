package details

import (
	AdviceRoutes "schoolku_backend/internals/features/home/advices/route"
	ArticleRoutes "schoolku_backend/internals/features/home/articles/route"
	FaqRoutes "schoolku_backend/internals/features/home/faqs/route"
	NotificationRoutes "schoolku_backend/internals/features/home/notifications/route"
	PostRoutes "schoolku_backend/internals/features/home/posts/route"
	QouteRoutes "schoolku_backend/internals/features/home/qoutes/route"
	QuestionnaireRoutes "schoolku_backend/internals/features/home/questionnaires/route"

	DBMiddleware "schoolku_backend/internals/middlewares"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// ✅ Untuk route publik tanpa token
// Contoh akses: /public/quotes
func HomePublicRoutes(api fiber.Router, db *gorm.DB) {
	// Route lainnya yang tidak membutuhkan DBMiddleware
	QouteRoutes.AllQuoteRoutes(api, db)
	FaqRoutes.AllFaqRoutes(api, db)
	ArticleRoutes.AllArticleRoutes(api, db)
	PostRoutes.AllPublicRoutes(api, db)
	QuestionnaireRoutes.AllQuestionnaireQuestionRoutes(api, db)

	// Hanya menambahkan DBMiddleware untuk DonationRoutes
	donationRoutes := api.Group("/payments")
	donationRoutes.Use(DBMiddleware.DBMiddleware(db)) // Apply DBMiddleware only for donation routes

}

// ✅ Untuk route user login (dengan token)
// Contoh akses: /api/u/notifications
func HomePrivateRoutes(api fiber.Router, db *gorm.DB) {
	AdviceRoutes.AllAdviceRoutes(api, db)

	FaqRoutes.FaqUserRoutes(api, db)
	PostRoutes.PostUserRoutes(api, db)
}

// ✅ Untuk route admin school (token + admin)
// Contoh akses: /api/a/quotes
func HomeAdminRoutes(api fiber.Router, db *gorm.DB) {
	NotificationRoutes.NotificationAdminRoutes(api, db)
	FaqRoutes.FaqAdminRoutes(api, db)
	AdviceRoutes.AdviceAdminRoutes(api, db)
	ArticleRoutes.ArticleAdminRoutes(api, db)
	PostRoutes.PostAdminRoutes(api, db)
	QuestionnaireRoutes.QuestionAdminRoutes(api, db)
	QouteRoutes.QuoteAdminRoutes(api, db)

}
