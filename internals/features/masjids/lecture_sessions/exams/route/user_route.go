package route

import (
	"masjidku_backend/internals/features/masjids/lecture_sessions/exams/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func LectureSessionsExamsUserRoutes(admin fiber.Router, db *gorm.DB) {
	examCtrl := controller.NewLectureSessionsExamController(db)
	userExamCtrl := controller.NewUserLectureSessionsExamController(db)

	// 📚 Group: /lecture-sessions-exams
	exam := admin.Group("/lecture-sessions-exams")
	exam.Post("/", examCtrl.CreateLectureSessionsExam)      // ➕ Buat ujian sesi kajian
	exam.Get("/", examCtrl.GetAllLectureSessionsExams)      // 📄 Lihat semua ujian
	exam.Get("/:id", examCtrl.GetLectureSessionsExamByID)   // 🔍 Detail ujian
	exam.Put("/:id", examCtrl.UpdateLectureSessionsExam)    // ✏️ Edit ujian
	exam.Delete("/:id", examCtrl.DeleteLectureSessionsExam) // ❌ Hapus ujian

	// 👥 Group: /user-lecture-sessions-exams
	userExam := admin.Group("/user-lecture-sessions-exams")
	userExam.Get("/", userExamCtrl.GetAllUserLectureSessionsExams)    // 📄 Lihat semua
	userExam.Get("/:id", userExamCtrl.GetUserLectureSessionsExamByID) // 🔍 Detail user ujian
}