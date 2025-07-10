package route

import (
	surveyController "masjidku_backend/internals/features/users/survey/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func SurveyUserRoutes(api fiber.Router, db *gorm.DB) {
	surveyQuestionCtrl := surveyController.NewSurveyQuestionController(db)
	userSurveyCtrl := surveyController.NewUserSurveyController(db)

	// 📋 Routes untuk pertanyaan survei (GET Only)
	surveyQuestionRoutes := api.Group("/survey-questions")
	surveyQuestionRoutes.Get("/", surveyQuestionCtrl.GetAll)
	surveyQuestionRoutes.Get("/:id", surveyQuestionCtrl.GetByID)

	// 📝 Routes untuk user menjawab survei
	userSurveyRoutes := api.Group("/user-surveys")
	userSurveyRoutes.Post("/", userSurveyCtrl.SubmitSurveyAnswers)
}
