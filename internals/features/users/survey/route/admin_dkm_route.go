package route

import (
	"madinahsalam_backend/internals/constants"
	surveyController "madinahsalam_backend/internals/features/users/survey/controller"
	authMiddleware "madinahsalam_backend/internals/middlewares/auth"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func SurveyAdminRoutes(api fiber.Router, db *gorm.DB) {
	surveyQuestionCtrl := surveyController.NewSurveyQuestionController(db)

	// ✅ Admin (admin/dkm) & Owner boleh kelola pertanyaan survei
	adminOrOwner := api.Group("/survey-questions",
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola pertanyaan survei"),
			constants.AdminAndAbove,
		),
	)

	adminOrOwner.Post("/", surveyQuestionCtrl.Create)
	adminOrOwner.Put("/:id", surveyQuestionCtrl.Update)
	adminOrOwner.Delete("/:id", surveyQuestionCtrl.Delete)

}
