package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	quizcontroller "masjidku_backend/internals/features/school/submissions_assesments/quizzes/controller"
)

/*
Catatan:
- Pasang middleware RequireTeacher di parent router `r` (prefix: /api/t).
- Path hasil:
  - /api/t/quizzes-teacher/...
  - /api/t/quiz-questions-teacher/...
  - (alias kompat) /api/t/quiz-items-teacher/...
  - /api/t/quizzes-teacher/attempt-answers-teacher/...
  - /api/t/quizzes-teacher/attempts-teacher/...
*/

func QuizzesTeacherRoutes(r fiber.Router, db *gorm.DB) {
	// ============================
	// QUIZZES (master) -> /api/t/quizzes-teacher
	// ============================
	quizCtrl := quizcontroller.NewQuizController(db)
	quizzes := r.Group("/quizzes-teacher")

	quizzes.Post("/",      quizCtrl.Create)   // POST   /api/t/quizzes-teacher
	quizzes.Patch("/:id",  quizCtrl.Patch)    // PATCH  /api/t/quizzes-teacher/:id
	quizzes.Delete("/:id", quizCtrl.Delete)   // DELETE /api/t/quizzes-teacher/:id

	// ============================
	// QUIZ QUESTIONS (soal & opsi JSONB) -> /api/t/quiz-questions-teacher
	// ============================
	qqCtrl := quizcontroller.NewQuizQuestionsController(db)
	qq := r.Group("/quiz-questions-teacher")

	// Rekomendasi expose full CRUD untuk kebutuhan guru
	qq.Get("/",      qqCtrl.List)    // GET    /api/t/quiz-questions-teacher?quiz_id=&type=&q=&page=&per_page=&sort=
	qq.Get("/list",  qqCtrl.List)    // alias
	qq.Post("/",     qqCtrl.Create)  // POST   /api/t/quiz-questions-teacher
	qq.Patch("/:id", qqCtrl.Patch)   // PATCH  /api/t/quiz-questions-teacher/:id
	qq.Delete("/:id", qqCtrl.Delete) // DELETE /api/t/quiz-questions-teacher/:id

	// ============================
	// USER QUIZ ATTEMPT ANSWERS (guru review/ubah)
	// child dari /quizzes-teacher
	// ============================
	uqaCtrl := quizcontroller.NewUserQuizAttemptAnswersController(db)
	ans := quizzes.Group("/user-quiz-attempts")

	ans.Get("/",       uqaCtrl.List)     // GET    /api/t/quizzes-teacher/attempt-answers-teacher?attempt_id=...&question_id=...
	ans.Post("/",      uqaCtrl.Create)   // POST   /api/t/quizzes-teacher/attempt-answers-teacher
	ans.Patch("/:id",  uqaCtrl.Patch)    // PATCH  /api/t/quizzes-teacher/attempt-answers-teacher/:id
	ans.Delete("/:id", uqaCtrl.Delete)   // DELETE /api/t/quizzes-teacher/attempt-answers-teacher/:id

	// ============================
	// USER QUIZ ATTEMPTS
	// child dari /quizzes-teacher
	// ============================
	uqAttemptCtrl := quizcontroller.NewUserQuizAttemptsController(db)
	attempts := quizzes.Group("/user-quiz-attempt-answers")

	attempts.Get("/",      uqAttemptCtrl.List)    // GET    /api/t/quizzes-teacher/attempts-teacher?quiz_id=&student_id=&status=&active_only=true
	attempts.Post("/",     uqAttemptCtrl.Create)  // POST   /api/t/quizzes-teacher/attempts-teacher
	attempts.Patch("/:id", uqAttemptCtrl.Patch)   // PATCH  /api/t/quizzes-teacher/attempts-teacher/:id
	attempts.Delete("/:id", uqAttemptCtrl.Delete) // DELETE /api/t/quizzes-teacher/attempts-teacher/:id
}
