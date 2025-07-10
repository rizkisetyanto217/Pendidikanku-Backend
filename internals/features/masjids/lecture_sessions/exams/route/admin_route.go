package route

import (
	"masjidku_backend/internals/features/masjids/lecture_sessions/exams/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func LectureSessionsExamsAdminRoutes(admin fiber.Router, db *gorm.DB) {
	ctrl := controller.NewLectureSessionsExamController(db)
	ctrl2 := controller.NewUserLectureSessionsExamController(db)

	// 📚 Group: /lecture-sessions-exams
	exam := admin.Group("/lecture-sessions-exams")
	exam.Post("/", ctrl.CreateLectureSessionsExam)      // ➕ Buat ujian sesi kajian
	exam.Get("/", ctrl.GetAllLectureSessionsExams)      // 📄 Lihat semua ujian
	exam.Get("/:id", ctrl.GetLectureSessionsExamByID)   // 🔍 Detail ujian
	exam.Put("/:id", ctrl.UpdateLectureSessionsExam)    // ✏️ Edit ujian
	exam.Delete("/:id", ctrl.DeleteLectureSessionsExam) // ❌ Hapus ujian

	// 👥 Group: /user-lecture-sessions-exams
	userExam := admin.Group("/user-lecture-sessions-exams")
	userExam.Get("/", ctrl2.GetAllUserLectureSessionsExams)    // 📄 Lihat semua
	userExam.Get("/:id", ctrl2.GetUserLectureSessionsExamByID) // 🔍 Detail user ujian
}