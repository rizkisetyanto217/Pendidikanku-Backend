package route

import (
	quizcontroller "schoolku_backend/internals/features/schools/lecture_sessions/quizzes/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Baca/kerjakan quiz tanpa login
func AllLectureSessionsQuizRoutes(api fiber.Router, db *gorm.DB) {
	quizCtrl := quizcontroller.NewLectureSessionsQuizController(db)
	pub := api.Group("/lecture-sessions-quiz")

	// Read-only publik
	pub.Get("/", quizCtrl.GetAllQuizzes)
	pub.Get("/by-school/:slug", quizCtrl.GetQuizzesBySlug)
	pub.Get("/:id", quizCtrl.GetQuizByID)
	pub.Get("/:id/with-questions", quizCtrl.GetByLectureSessionID)
	pub.Get("/:slug/with-questions-by-slug", quizCtrl.GetByLectureSessionSlug)
	pub.Get("/by-lecture/:id", quizCtrl.GetByLectureID)
	pub.Get("/by-lecture-slug/:lecture_slug", quizCtrl.GetQuizzesByLectureSlug)

	// ✅ Submit hasil quiz TANPA login (pakai controller yang sama)
	userQuizCtrl := quizcontroller.NewUserLectureSessionsQuizController(db)
	pubUser := api.Group("/user-lecture-sessions-quiz/public")
	// pastikan param bernama "lecture_session_slug" (sesuai controller)
	pubUser.Post("/by-session/:lecture_session_slug", userQuizCtrl.CreateUserLectureSessionsQuiz)
	// opsi alias (kalau mau path lebih pendek)
	pubUser.Post("/:lecture_session_slug", userQuizCtrl.CreateUserLectureSessionsQuiz)
}
