package route

import (
	"schoolku_backend/internals/constants"
	surveyController "schoolku_backend/internals/features/users/survey/controller"
	authMiddleware "schoolku_backend/internals/middlewares/auth"

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
