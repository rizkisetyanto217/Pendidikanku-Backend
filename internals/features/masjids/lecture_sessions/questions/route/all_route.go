package route

import (
	"masjidku_backend/internals/features/masjids/lecture_sessions/questions/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AllLectureSessionsQuestionRoutes(user fiber.Router, db *gorm.DB) {
	questionCtrl := controller.NewLectureSessionsQuestionController(db)
	// 📝 Group: /lecture-sessions-questions (read-only)
	questions := user.Group("/lecture-sessions-questions")
	questions.Get("/", questionCtrl.GetAllLectureSessionsQuestions)
	// questions.Get("/:id", questionCtrl.GetLectureSessionsQuestionByID) // (opsional)
}