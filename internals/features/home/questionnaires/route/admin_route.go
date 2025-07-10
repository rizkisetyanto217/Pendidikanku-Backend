package route

import (
	"masjidku_backend/internals/features/home/questionnaires/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func QuestionnaireQuestionAdminRoutes(api fiber.Router, db *gorm.DB) {
	ctrl := controller.NewQuestionnaireQuestionController(db)

	// 👤 User routes (read only)
	user := api.Group("/questionnaires")
	user.Post("/", ctrl.CreateQuestion)           // 🔍 Detail pertanyaan
          // ❌ Hapus pertanyaan kuisioner
}
