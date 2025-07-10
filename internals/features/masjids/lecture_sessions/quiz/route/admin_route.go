package route

import (
	quizcontroller "masjidku_backend/internals/features/masjids/lecture_sessions/quiz/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func LectureSessionsQuizAdminRoutes(admin fiber.Router, db *gorm.DB) {

	quizCtrl := quizcontroller.NewLectureSessionsQuizController(db)
	quizzes := admin.Group("/lecture-sessions-quiz")
	quizzes.Post("/", quizCtrl.CreateQuiz)          // ➕ Tambah quiz
	quizzes.Get("/", quizCtrl.GetAllQuizzes)        // 📄 Lihat semua quiz
	quizzes.Get("/:id", quizCtrl.GetQuizByID)       // 🔍 Lihat detail quiz
	quizzes.Delete("/:id", quizCtrl.DeleteQuizByID) // ❌ Hapus quiz

	userQuizCtrl := quizcontroller.NewUserLectureSessionsQuizController(db)
	userQuiz := admin.Group("/user-lecture-sessions-quiz")
	userQuiz.Post("/", userQuizCtrl.CreateUserLectureSessionsQuiz)           // ➕ Submit nilai quiz
	userQuiz.Get("/filter", userQuizCtrl.GetUserLectureSessionsQuizFiltered) // 🔍 Lihat hasil quiz user tertentu

}