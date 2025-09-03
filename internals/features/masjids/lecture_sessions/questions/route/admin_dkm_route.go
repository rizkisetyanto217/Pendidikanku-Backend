package route

import (
	"masjidku_backend/internals/constants"
	questionController "masjidku_backend/internals/features/masjids/lecture_sessions/questions/controller"
	authMiddleware "masjidku_backend/internals/middlewares/auth"
	masjidkuMiddleware "masjidku_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// 🔐 Admin/DKM/Owner (CRUD soal, hapus jawaban user)
func LectureSessionsQuestionAdminRoutes(router fiber.Router, db *gorm.DB) {
	questionCtrl := questionController.NewLectureSessionsQuestionController(db)
	userQuestionCtrl := questionController.NewLectureSessionsUserQuestionController(db)

	// Group besar: login + role admin/dkm/owner + scope masjid
	adminOrOwner := router.Group("/",
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola soal & jawaban ujian sesi"),
			constants.AdminAndAbove, // admin, dkm, owner
		),
		masjidkuMiddleware.IsMasjidAdmin(), // inject masjid_id scope
	)

	// 📝 /lecture-sessions-questions (CRUD)
	questions := adminOrOwner.Group("/lecture-sessions-questions")
	questions.Post("/", questionCtrl.CreateLectureSessionsQuestion)   // ➕ Tambah soal
	questions.Get("/", questionCtrl.GetAllLectureSessionsQuestions)   // 📄 Semua soal (scoped)
	questions.Put("/:id", questionCtrl.UpdateLectureSessionsQuestionByID) // ✏️ Ubah soal
	// questions.Get("/:id", questionCtrl.GetLectureSessionsQuestionByID) // (opsional)
	questions.Delete("/:id", questionCtrl.DeleteLectureSessionsQuestion)  // ❌ Hapus soal

	// 👤 /lecture-sessions-user-questions (admin bisa hapus jawaban user)
	userQuestions := adminOrOwner.Group("/lecture-sessions-user-questions")
	userQuestions.Delete("/:id", userQuestionCtrl.DeleteByID) // ❌ Hapus jawaban user
}
