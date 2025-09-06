package details

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	teacherProfile "masjidku_backend/internals/features/lembaga/teachers_students/route"
	surveyRoute "masjidku_backend/internals/features/users/survey/route"
	userRoute "masjidku_backend/internals/features/users/user/route"
	rateLimiter "masjidku_backend/internals/middlewares"
)

func UserRoutes(app *fiber.App, db *gorm.DB) {
	api := app.Group("/api",
		rateLimiter.GlobalRateLimiter(),
	)

	adminGroup := api.Group("/a") // 🔐 hanya teacher/admin/owner
	userRoute.UserAdminRoutes(adminGroup, db)
	surveyRoute.SurveyAdminRoutes(adminGroup, db)

	// 🔓 Prefix user biasa: /api/u/...
	userGroup := api.Group("/u") // 👤 user login biasa
	userRoute.UserUserRoutes(userGroup, db)
	surveyRoute.SurveyUserRoutes(userGroup, db)
	// teacherProfile.UsersTeacherTeacherRoute(userGroup, db)
	teacherProfile.UsersTeacherUserRoute(userGroup, db)
	
}