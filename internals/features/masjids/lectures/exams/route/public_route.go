package route

import (
	"masjidku_backend/internals/features/masjids/lectures/exams/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func LectureExamsUserRoutes(public fiber.Router, db *gorm.DB) {
	examCtrl := controller.NewLectureExamController(db)
	userExamCtrl := controller.NewUserLectureExamController(db)

	// 📚 Group: /lecture-exams
	exam := public.Group("/lecture-exams")
	exam.Get("/:id/questions", examCtrl.GetLectureExamWithQuestions)
	exam.Get("/:id/questions/by-lecture", examCtrl.GetQuestionExamByLectureID)
	exam.Post("/", examCtrl.CreateLectureExam)      // ➕ Buat ujian sesi kajian
	exam.Get("/", examCtrl.GetAllLectureExams)      // 📄 Lihat semua ujian
	exam.Get("/:id", examCtrl.GetLectureExamByID)   // 🔍 Detail ujian
	exam.Delete("/:id", examCtrl.DeleteLectureExam) // ❌ Hapus ujian
	exam.Put("/:id", examCtrl.UpdateLectureExam)    // ✏️ Edit ujian

	// 👥 Group: /user-lecture--exams
	userExam := public.Group("/user-lecture-exams")
	userExam.Get("/", userExamCtrl.GetAllUserLectureExams)    // 📄 Lihat semua
	userExam.Get("/:id", userExamCtrl.GetUserLectureExamByID) // 🔍 Detail user ujian

	// POST - User submit hasil exam (progress)
	userExam.Post("/", userExamCtrl.CreateUserLectureExam)
}