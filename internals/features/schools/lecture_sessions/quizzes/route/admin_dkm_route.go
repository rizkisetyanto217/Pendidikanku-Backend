package route

import (
	"schoolku_backend/internals/constants"
	quizcontroller "schoolku_backend/internals/features/schools/lecture_sessions/quizzes/controller"
	authMiddleware "schoolku_backend/internals/middlewares/auth"

	// schoolkuMiddleware "schoolku_backend/internals/middlewares/features" // aktifkan jika perlu scope school

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// CUD + laporan internal
func LectureSessionsQuizAdminRoutes(api fiber.Router, db *gorm.DB) {
	admin := api.Group("/",
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola quiz"),
			constants.AdminAndAbove,
		),
		// schoolkuMiddleware.IsSchoolAdmin(),
	)

	quizCtrl := quizcontroller.NewLectureSessionsQuizController(db)
	quizzes := admin.Group("/lecture-sessions-quiz")
	quizzes.Post("/", quizCtrl.CreateQuiz)
	quizzes.Get("/", quizCtrl.GetAllQuizzes)
	quizzes.Get("/get-by-id/:id", quizCtrl.GetQuizByID)
	quizzes.Get("/by-school", quizCtrl.GetQuizzesBySchoolID)
	quizzes.Put("/:id", quizCtrl.UpdateQuizByID)
	quizzes.Delete("/:id", quizCtrl.DeleteQuizByID)

	userQuizCtrl := quizcontroller.NewUserLectureSessionsQuizController(db)
	userQuiz := admin.Group("/user-lecture-sessions-quiz")
	userQuiz.Post("/", userQuizCtrl.CreateUserLectureSessionsQuiz)           // laporan/rekap by admin
	userQuiz.Get("/filter", userQuizCtrl.GetUserLectureSessionsQuizFiltered) // laporan internal
}
