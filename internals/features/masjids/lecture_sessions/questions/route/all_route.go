package route

import (
	questionController "masjidku_backend/internals/features/masjids/lecture_sessions/questions/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// 👥 All/User (read-only) – publik; kalau perlu login, tambahkan AuthMiddleware(db).
func AllLectureSessionsQuestionRoutes(router fiber.Router, db *gorm.DB) {
	questionCtrl := questionController.NewLectureSessionsQuestionController(db)

	// 📝 /lecture-sessions-questions (read-only)
	questions := router.Group("/lecture-sessions-questions")
	questions.Get("/", questionCtrl.GetAllLectureSessionsQuestions)             // 📄 Semua soal
	questions.Get("/by-quiz/:quiz_id", questionCtrl.GetLectureSessionsQuestionsByQuizID) // 🔎 Soal per quiz
	// questions.Get("/:id", questionCtrl.GetLectureSessionsQuestionByID) // (opsional)
}
