package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	quizQuestionsController "madinahsalam_backend/internals/features/school/submissions_assesments/quizzes/controller/questions"
	quizzesController "madinahsalam_backend/internals/features/school/submissions_assesments/quizzes/controller/quizzes"
	studentAttemptsController "madinahsalam_backend/internals/features/school/submissions_assesments/quizzes/controller/student_attempts"
)

/*
Catatan:
- Pasang middleware RequireTeacher di parent router `r` (prefix: /api/t).
- School context diambil dari ResolveSchoolContext (token / dsb), bukan dari path.
- Path hasil:
  - /api/t/quizzes-teacher/...
  - /api/t/quiz-questions-teacher/...
  - (alias) /api/t/quiz-items-teacher/...
  - /api/t/quizzes-teacher/attempt-answers-teacher/...
  - /api/t/quizzes-teacher/attempts-teacher/...
*/

func QuizzesTeacherRoutes(r fiber.Router, db *gorm.DB) {
	// Langsung mount di base /api/t
	mountQuizTeacherRoutes(r, db)
}

func mountQuizTeacherRoutes(base fiber.Router, db *gorm.DB) {
	// ============================
	// QUIZZES (master) -> /api/t/quizzes-teacher
	// ============================
	quizCtrl := quizzesController.NewQuizController(db)
	quizzes := base.Group("/quizzes-teacher")

	quizzes.Post("/", quizCtrl.Create)      // POST   /api/t/quizzes-teacher
	quizzes.Patch("/:id", quizCtrl.Patch)   // PATCH  /api/t/quizzes-teacher/:id
	quizzes.Delete("/:id", quizCtrl.Delete) // DELETE /api/t/quizzes-teacher/:id

	// ============================
	// QUIZ QUESTIONS (soal & opsi JSONB)
	// -> /api/t/quiz-questions-teacher  (+ alias /api/t/quiz-items-teacher)
	// ============================
	qqCtrl := quizQuestionsController.NewQuizQuestionsController(db)
	qqMain := base.Group("/quiz-questions-teacher")
	mountQuizQuestionsGroup(qqMain, qqCtrl)

	// Alias kompatibel
	qqAlias := base.Group("/quiz-items-teacher")
	mountQuizQuestionsGroup(qqAlias, qqCtrl)

	// ============================
	// USER QUIZ ATTEMPTS
	// child dari /quizzes-teacher -> /attempts-teacher
	// ============================
	uqAttemptCtrl := studentAttemptsController.NewStudentQuizAttemptsController(db)
	attempts := quizzes.Group("/attempts-teacher")

	attempts.Get("/", uqAttemptCtrl.List)         // GET    /api/t/quizzes-teacher/attempts-teacher?quiz_id=&student_id=&status=&active_only=true
	attempts.Post("/", uqAttemptCtrl.Create)      // POST   /api/t/quizzes-teacher/attempts-teacher
	attempts.Patch("/:id", uqAttemptCtrl.Patch)   // PATCH  /api/t/quizzes-teacher/attempts-teacher/:id
	attempts.Delete("/:id", uqAttemptCtrl.Delete) // DELETE /api/t/quizzes-teacher/attempts-teacher/:id
}

// Hindari duplikasi handler antara quiz-questions-teacher dan alias quiz-items-teacher
func mountQuizQuestionsGroup(g fiber.Router, qqCtrl *quizQuestionsController.QuizQuestionsController) {
	g.Get("/", qqCtrl.List)         // GET    /.../quiz-questions-teacher?quiz_id=&type=&q=&page=&per_page=&sort=
	g.Get("/list", qqCtrl.List)     // alias
	g.Post("/", qqCtrl.Create)      // POST   /.../quiz-questions-teacher
	g.Patch("/:id", qqCtrl.Patch)   // PATCH  /.../quiz-questions-teacher/:id
	g.Delete("/:id", qqCtrl.Delete) // DELETE /.../quiz-questions-teacher/:id
}
