package route

import (
	linkcontroller "masjidku_backend/internals/features/masjids/lecture_sessions/questions/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func LectureSessionsQuestionAdminRoutes(admin fiber.Router, db *gorm.DB) {
	questionCtrl := linkcontroller.NewLectureSessionsQuestionController(db)
	userQuestionCtrl := linkcontroller.NewLectureSessionsUserQuestionController(db)

	// 📝 Group: /lecture-sessions-questions
	questions := admin.Group("/lecture-sessions-questions")
	questions.Post("/", questionCtrl.CreateLectureSessionsQuestion) // ➕ Tambah soal
	questions.Get("/", questionCtrl.GetAllLectureSessionsQuestions) // 📄 Lihat semua soal
	// questions.Get("/:id", questionCtrl.GetLectureSessionsQuestionByID) // 🔍 (jika diperlukan)
	questions.Delete("/:id", questionCtrl.DeleteLectureSessionsQuestion) // ❌ Hapus soal

	// 👤 Group: /lecture-sessions-user-questions
	userQuestions := admin.Group("/lecture-sessions-user-questions")
	userQuestions.Delete("/:id", userQuestionCtrl.DeleteByID) // ❌ Hapus jawaban user

}
