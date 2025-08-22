package route

import (
	"masjidku_backend/internals/constants"
	quizcontroller "masjidku_backend/internals/features/masjids/lecture_sessions/quizzes/controller"
	authMiddleware "masjidku_backend/internals/middlewares/auth"

	// masjidkuMiddleware "masjidku_backend/internals/middlewares/features" // aktifkan jika perlu scope masjid

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// CUD + laporan internal
func LectureSessionsQuizAdminRoutes(api fiber.Router, db *gorm.DB) {
	admin := api.Group("/",
		authMiddleware.AuthMiddleware(db),
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola quiz"),
			constants.AdminAndAbove,
		),
		// masjidkuMiddleware.IsMasjidAdmin(),
	)

	quizCtrl := quizcontroller.NewLectureSessionsQuizController(db)
	quizzes := admin.Group("/lecture-sessions-quiz")
	quizzes.Post("/", quizCtrl.CreateQuiz)
	quizzes.Get("/", quizCtrl.GetAllQuizzes)
	quizzes.Get("/get-by-id/:id", quizCtrl.GetQuizByID)
	quizzes.Get("/by-masjid", quizCtrl.GetQuizzesByMasjidID)
	quizzes.Put("/:id", quizCtrl.UpdateQuizByID)
	quizzes.Delete("/:id", quizCtrl.DeleteQuizByID)

	userQuizCtrl := quizcontroller.NewUserLectureSessionsQuizController(db)
	userQuiz := admin.Group("/user-lecture-sessions-quiz")
	userQuiz.Post("/", userQuizCtrl.CreateUserLectureSessionsQuiz)           // laporan/rekap by admin
	userQuiz.Get("/filter", userQuizCtrl.GetUserLectureSessionsQuizFiltered) // laporan internal
}
