package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	quizcontroller "madinahsalam_backend/internals/features/school/submissions_assesments/quizzes/controller"
)

/*
Catatan:
- Base (user/public): /api/u
- Controller sisi user WAJIB filter quizzes_is_published = true (di-controller, bukan di route).
- School context diambil via ResolveSchoolContext (token / slug / dll), bukan path.
- Path utama (baru):
  - /api/u/quizzes/...
  - /api/u/quiz-questions/...
  - /api/u/quizzes/attempts/...
  - /api/u/quizzes/attempt-answers/...
- Alias kompatibel:
  - /api/u/user-quiz-attempts/...
  - /api/u/user-quiz-attempt-answers/...
*/

func QuizzesUserRoutes(r fiber.Router, db *gorm.DB) {
	// Langsung mount di base /api/u
	mountQuizUserRoutes(r, db)
}

func mountQuizUserRoutes(base fiber.Router, db *gorm.DB) {
	// ============================
	// QUIZZES (read-only publik)
	// ============================
	quizCtrl := quizcontroller.NewQuizController(db)
	quizzes := base.Group("/quizzes") // -> /api/u/quizzes

	quizzes.Get("/", quizCtrl.List)     // GET /api/u/quizzes
	quizzes.Get("/list", quizCtrl.List) // alias

	// ============================
	// QUIZ QUESTIONS (read-only)
	// ============================
	qqCtrl := quizcontroller.NewQuizQuestionsController(db)
	qq := base.Group("/quiz-questions") // -> /api/u/quiz-questions

	qq.Get("/", qqCtrl.List)     // GET /api/u/quiz-questions?quiz_id=&type=&q=&page=&per_page=&sort=
	qq.Get("/list", qqCtrl.List) // alias

	// ============================
	// USER QUIZ ATTEMPTS (user)
	// nested di /quizzes + alias kompatibel
	// ============================
	uqAttemptCtrl := quizcontroller.NewStudentQuizAttemptsController(db)

	// Nested utama
	attempts := quizzes.Group("/attempts") // -> /api/u/quizzes/attempts
	mountUserAttemptsGroup(attempts, uqAttemptCtrl)

	// Alias kompatibel (rute lama)
	attemptsAlias := base.Group("/user-quiz-attempts") // -> /api/u/user-quiz-attempts
	mountUserAttemptsGroup(attemptsAlias, uqAttemptCtrl)

}

// Hindari duplikasi handler untuk attempts (nested & alias)
func mountUserAttemptsGroup(g fiber.Router, ctrl *quizcontroller.StudentQuizAttemptsController) {
	g.Get("/", ctrl.List)         // GET list
	g.Get("/list", ctrl.List)     // alias
	g.Post("/", ctrl.Create)      // POST create attempt
	g.Patch("/:id", ctrl.Patch)   // PATCH attempt by id
	g.Delete("/:id", ctrl.Delete) // DELETE attempt by id
}
