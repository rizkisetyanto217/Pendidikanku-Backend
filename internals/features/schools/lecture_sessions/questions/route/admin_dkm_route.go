package route

import (
	"schoolku_backend/internals/constants"
	questionController "schoolku_backend/internals/features/schools/lecture_sessions/questions/controller"
	authMiddleware "schoolku_backend/internals/middlewares/auth"
	schoolkuMiddleware "schoolku_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// 🔐 Admin/DKM/Owner (CRUD soal, hapus jawaban user)
func LectureSessionsQuestionAdminRoutes(router fiber.Router, db *gorm.DB) {
	questionCtrl := questionController.NewLectureSessionsQuestionController(db)
	userQuestionCtrl := questionController.NewLectureSessionsUserQuestionController(db)

	// Group besar: login + role admin/dkm/owner + scope school
	adminOrOwner := router.Group("/",
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola soal & jawaban ujian sesi"),
			constants.AdminAndAbove, // admin, dkm, owner
		),
		schoolkuMiddleware.IsSchoolAdmin(), // inject school_id scope
	)

	// 📝 /lecture-sessions-questions (CRUD)
	questions := adminOrOwner.Group("/lecture-sessions-questions")
	questions.Post("/", questionCtrl.CreateLectureSessionsQuestion)       // ➕ Tambah soal
	questions.Get("/", questionCtrl.GetAllLectureSessionsQuestions)       // 📄 Semua soal (scoped)
	questions.Put("/:id", questionCtrl.UpdateLectureSessionsQuestionByID) // ✏️ Ubah soal
	// questions.Get("/:id", questionCtrl.GetLectureSessionsQuestionByID) // (opsional)
	questions.Delete("/:id", questionCtrl.DeleteLectureSessionsQuestion) // ❌ Hapus soal

	// 👤 /lecture-sessions-user-questions (admin bisa hapus jawaban user)
	userQuestions := adminOrOwner.Group("/lecture-sessions-user-questions")
	userQuestions.Delete("/:id", userQuestionCtrl.DeleteByID) // ❌ Hapus jawaban user
}
