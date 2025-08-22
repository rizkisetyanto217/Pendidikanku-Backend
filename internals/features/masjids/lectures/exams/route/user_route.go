package route

import (
	examController "masjidku_backend/internals/features/masjids/lectures/exams/controller"
	authMiddleware "masjidku_backend/internals/middlewares/auth"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// 🌐 User (terkait progress & sertifikat) – login wajib, tanpa guard role
func LectureExamsUserRoutes(router fiber.Router, db *gorm.DB) {
	userExamCtrl := examController.NewUserLectureExamController(db)

	userExam := router.Group("/user-lecture-exams",
		authMiddleware.AuthMiddleware(db), // user harus login
	)

	userExam.Get("/", userExamCtrl.GetAllUserLectureExams)    // 📄 Riwayat ujian user (by current user)
	userExam.Get("/:id", userExamCtrl.GetUserLectureExamByID) // 🔍 Detail hasil ujian user
	userExam.Post("/", userExamCtrl.CreateUserLectureExam)    // 📝 Submit hasil ujian (progress untuk sertif)
}
