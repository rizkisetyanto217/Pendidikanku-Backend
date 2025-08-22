package route

import (
	"masjidku_backend/internals/features/home/questionnaires/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AllQuestionnaireQuestionRoutes(api fiber.Router, db *gorm.DB) {
	ctrl := controller.NewQuestionnaireQuestionController(db)

	// 🛡️ Admin routes (manage data)
	admin := api.Group("/questionnaires")
	admin.Post("/", ctrl.CreateQuestion)

	ctrl2 := controller.NewUserQuestionnaireAnswerController(db)

	user := api.Group("/user-questionnaires")
	user.Post("/", ctrl2.SubmitBulkAnswers)                   // ✅ Submit jawaban batch/
}
