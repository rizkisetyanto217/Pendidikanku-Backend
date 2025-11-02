package route

import (
	"schoolku_backend/internals/constants"
	quizcontroller "schoolku_backend/internals/features/schools/lecture_sessions/quizzes/controller"
	authMiddleware "schoolku_backend/internals/middlewares/auth"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Aksi/riwayat milik user login
func LectureSessionsQuizUserRoutes(api fiber.Router, db *gorm.DB) {
	user := api.Group("/",
		authMiddleware.OnlyRolesSlice(
			"❌ Hanya pengguna terautentikasi yang boleh mengakses fitur quiz (user).",
			constants.AllowedRoles,
		),
	)

	quizCtrl := quizcontroller.NewLectureSessionsQuizController(db)
	quizzes := user.Group("/lecture-sessions-quiz")
	quizzes.Get("/", quizCtrl.GetAllQuizzes)
	quizzes.Get("/by-school/:slug", quizCtrl.GetQuizzesBySlug)
	quizzes.Get("/:id", quizCtrl.GetQuizByID)
	quizzes.Get("/:id/with-questions", quizCtrl.GetByLectureSessionID)
	quizzes.Get("/:slug/with-questions-by-slug", quizCtrl.GetByLectureSessionSlug)
	quizzes.Get("/by-lecture/:id", quizCtrl.GetByLectureID)
	quizzes.Get("/by-lecture-slug/:lecture_slug", quizCtrl.GetQuizzesByLectureSlug)

	userQuizCtrl := quizcontroller.NewUserLectureSessionsQuizController(db)
	userQuiz := user.Group("/user-lecture-sessions-quiz")
	userQuiz.Post("/:slug", userQuizCtrl.CreateUserLectureSessionsQuiz) // submit (login)
	userQuiz.Post("/by-session/:lecture_session_slug", userQuizCtrl.CreateUserLectureSessionsQuiz)
	userQuiz.Get("/", userQuizCtrl.GetAllUserLectureSessionsQuiz)            // riwayat milik user
	userQuiz.Get("/filter", userQuizCtrl.GetUserLectureSessionsQuizFiltered) // filter milik user
	userQuiz.Delete("/:id", userQuizCtrl.DeleteUserLectureSessionsQuizByID)  // hapus milik user
	userQuiz.Get("/with-detail", userQuizCtrl.GetUserQuizWithDetail)         // detail riwayat milik user
}
