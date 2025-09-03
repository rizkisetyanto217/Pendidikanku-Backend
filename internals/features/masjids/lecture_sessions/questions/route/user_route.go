package route

import (
	questionController "masjidku_backend/internals/features/masjids/lecture_sessions/questions/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// 👤 User (submit & baca jawaban milik sendiri)
func LectureSessionsQuestionUserRoutes(router fiber.Router, db *gorm.DB) {
	userQuestionCtrl := questionController.NewLectureSessionsUserQuestionController(db)

	// Login wajib, tanpa guard role
	userQuestions := router.Group("/lecture-sessions-user-questions",

	)

	userQuestions.Post("/", userQuestionCtrl.CreateLectureSessionsUserQuestion) // 📝 Submit jawaban
	// userQuestions.Get("/", userQuestionCtrl.GetAllUserLectureSessionsQuestions) // (opsional: list jawaban user)
	userQuestions.Get("/by-question/:question_id", userQuestionCtrl.GetByQuestionID) // 🔎 Jawaban user by question
	// userQuestions.Get("/:id", userQuestionCtrl.GetLectureSessionsUserQuestionByID)  // (opsional)
}
