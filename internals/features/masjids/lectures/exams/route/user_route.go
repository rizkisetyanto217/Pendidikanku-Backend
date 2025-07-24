package route

import (
	"masjidku_backend/internals/features/masjids/lectures/exams/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func LectureExamsUserRoutes(admin fiber.Router, db *gorm.DB) {
	examCtrl := controller.NewLectureExamController(db)
	userExamCtrl := controller.NewUserLectureExamController(db)

	// 📚 Group: /lecture--exams
	exam := admin.Group("/lecture-exams")
	exam.Post("/", examCtrl.CreateLectureExam)      // ➕ Buat ujian sesi kajian
	exam.Get("/", examCtrl.GetAllLectureExams)      // 📄 Lihat semua ujian
	exam.Get("/:id", examCtrl.GetLectureExamByID)   // 🔍 Detail ujian
	exam.Put("/:id", examCtrl.UpdateLectureExam)    // ✏️ Edit ujian
	exam.Delete("/:id", examCtrl.DeleteLectureExam) // ❌ Hapus ujian

	// 👥 Group: /user-lecture--exams
	userExam := admin.Group("/user-lecture-exams")
	userExam.Get("/", userExamCtrl.GetAllUserLectureExams)    // 📄 Lihat semua
	userExam.Get("/:id", userExamCtrl.GetUserLectureExamByID) // 🔍 Detail user ujian
}