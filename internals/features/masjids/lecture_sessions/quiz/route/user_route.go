package route

import (
	quizcontroller "masjidku_backend/internals/features/masjids/lecture_sessions/quiz/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func LectureSessionsQuizUserRoutes(user fiber.Router, db *gorm.DB) {
	quizCtrl := quizcontroller.NewLectureSessionsQuizController(db)

	quizzes := user.Group("/lecture-sessions-quiz")
	quizzes.Get("/", quizCtrl.GetAllQuizzes)  // 📄 Lihat semua quiz
	quizzes.Get("/:id", quizCtrl.GetQuizByID) // 🔍 Lihat detail quiz

	userQuizCtrl := quizcontroller.NewUserLectureSessionsQuizController(db)

	userQuiz := user.Group("/user-lecture-sessions-quiz")
	userQuiz.Post("/", userQuizCtrl.CreateUserLectureSessionsQuiz)           // ➕ Input hasil quiz user
	userQuiz.Get("/", userQuizCtrl.GetAllUserLectureSessionsQuiz)            // 📄 Lihat semua hasil quiz user
	userQuiz.Get("/filter", userQuizCtrl.GetUserLectureSessionsQuizFiltered) // 🔍 Filter by quiz_id/user_id
	userQuiz.Delete("/:id", userQuizCtrl.DeleteUserLectureSessionsQuizByID)  // ❌ Hapus hasil quiz
	userQuiz.Get("/with-detail", userQuizCtrl.GetUserQuizWithDetail)
}
