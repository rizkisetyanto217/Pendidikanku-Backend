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
func QuestionAdminRoutes(router fiber.Router, db *gorm.DB) {
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

	// 📝 /lecture-sessions-questions (CRUD + read by session)
	questions := adminOrOwner.Group("/lecture-sessions-questions")
	questions.Post("/", questionCtrl.CreateLectureSessionsQuestion)       // ➕ Tambah soal
	questions.Get("/", questionCtrl.GetAllLectureSessionsQuestions)       // 📄 Semua soal (scoped)
	questions.Get("/:id", questionCtrl.GetLectureSessionsQuestionByID)    // 🔍 Detail soal (diaktifkan)
	questions.Put("/:id", questionCtrl.UpdateLectureSessionsQuestionByID) // ✏️ Ubah soal
	questions.Delete("/:id", questionCtrl.DeleteLectureSessionsQuestion)  // ❌ Hapus soal

	// (opsional) list soal per sesi — uncomment jika controllernya ada
	// questions.Get("/by-session/:lecture_session_id", questionCtrl.GetLectureSessionsQuestionsBySessionID)
	// questions.Get("/by-session-slug/:lecture_session_slug", questionCtrl.GetLectureSessionsQuestionsBySessionSlug)

	// 👤 /lecture-sessions-user-questions (admin bisa hapus jawaban user)
	userQuestions := adminOrOwner.Group("/lecture-sessions-user-questions")
	userQuestions.Delete("/:id", userQuestionCtrl.DeleteByID) // ❌ Hapus jawaban user

	// (opsional) laporan/insight jawaban user — uncomment jika controllernya ada
	// userQuestions.Get("/", userQuestionCtrl.GetAll)                                // list semua jawaban (scoped)
	// userQuestions.Get("/filter", userQuestionCtrl.GetFiltered)                     // ?user_id=&lecture_session_id=
	// userQuestions.Get("/by-session/:lecture_session_id", userQuestionCtrl.GetBySessionID)
	// userQuestions.Get("/by-session-slug/:lecture_session_slug", userQuestionCtrl.GetBySessionSlug)
	// userQuestions.Delete("/by-user/:user_id/by-session/:lecture_session_id", userQuestionCtrl.DeleteByUserAndSession)
}
