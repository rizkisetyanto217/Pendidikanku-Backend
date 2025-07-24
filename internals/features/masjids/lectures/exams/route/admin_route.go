package route

import (
	"masjidku_backend/internals/features/masjids/lectures/exams/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func LectureExamsAdminRoutes(admin fiber.Router, db *gorm.DB) {
	ctrl := controller.NewLectureExamController(db)
	ctrl2 := controller.NewUserLectureExamController(db)

	// 📚 Group: /lecture--exams
	exam := admin.Group("/lecture-exams")
	exam.Post("/", ctrl.CreateLectureExam)      // ➕ Buat ujian sesi kajian
	exam.Get("/", ctrl.GetAllLectureExams)      // 📄 Lihat semua ujian
	exam.Get("/:id", ctrl.GetLectureExamByID)   // 🔍 Detail ujian
	exam.Put("/:id", ctrl.UpdateLectureExam)    // ✏️ Edit ujian
	exam.Delete("/:id", ctrl.DeleteLectureExam) // ❌ Hapus ujian

	// 👥 Group: /user-lecture--exams
	userExam := admin.Group("/user-lecture-exams")
	userExam.Get("/", ctrl2.GetAllUserLectureExams)    // 📄 Lihat semua
	userExam.Get("/:id", ctrl2.GetUserLectureExamByID) // 🔍 Detail user ujian
}