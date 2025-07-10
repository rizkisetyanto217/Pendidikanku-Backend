package details

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	surveyRoute "masjidku_backend/internals/features/users/survey/route"
	tokenRoute "masjidku_backend/internals/features/users/token/route"
	userRoute "masjidku_backend/internals/features/users/user/routes"
	rateLimiter "masjidku_backend/internals/middlewares"
	authMiddleware "masjidku_backend/internals/middlewares/auth"
)

func UserRoutes(app *fiber.App, db *gorm.DB) {
	api := app.Group("/api",
		authMiddleware.AuthMiddleware(db),
		rateLimiter.GlobalRateLimiter(),
	)

	adminGroup := api.Group("/a") // 🔐 hanya teacher/admin/owner
	userRoute.UserAdminRoutes(adminGroup, db)
	surveyRoute.SurveyAdminRoutes(adminGroup, db)

	// 🔓 Prefix user biasa: /api/u/...
	userGroup := api.Group("/u") // 👤 user login biasa
	userRoute.UserAllRoutes(userGroup, db)
	surveyRoute.SurveyUserRoutes(userGroup, db)
	tokenRoute.RegisterTokenRoutes(userGroup, db)   // 🔓 Token routes
}
