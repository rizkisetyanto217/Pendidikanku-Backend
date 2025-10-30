// file: internals/route/details/user_routes.go (atau sesuai lokasinya)
package details

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	teacherProfile "masjidku_backend/internals/features/lembaga/masjid_yayasans/teachers_students/route"
	surveyRoute "masjidku_backend/internals/features/users/survey/route"
	userTeacher "masjidku_backend/internals/features/users/user_teachers/route"
	userRoute "masjidku_backend/internals/features/users/users/route"
	rateLimiter "masjidku_backend/internals/middlewares"
)

func UserRoutes(app *fiber.App, db *gorm.DB) {
	// === PUBLIC ===
	public := app.Group("/api/public",
		rateLimiter.GlobalRateLimiter(),
	)
	// 🔓 Jadikan daftar/gateway teacher publik
	teacherProfile.AllTeacherUserRoute(public, db)

	// === API (protected / structured) ===
	api := app.Group("/api",
		rateLimiter.GlobalRateLimiter(),
	)

	// 🔒 ADMIN
	adminGroup := api.Group("/a")
	userRoute.UserAdminRoutes(adminGroup, db)
	surveyRoute.SurveyAdminRoutes(adminGroup, db)

	// 👤 USER LOGIN (tetap)
	userGroup := api.Group("/u")
	userRoute.UserUserRoutes(userGroup, db)
	surveyRoute.SurveyUserRoutes(userGroup, db)

	// (tetap) route lain yg memang butuh login
	userTeacher.UserTeachersRoute(userGroup, db)

	// ❌ HAPUS/PASTIKAN TIDAK ADA LAGI:
	// teacherProfile.AllTeacherUserRoute(userGroup, db)
}
