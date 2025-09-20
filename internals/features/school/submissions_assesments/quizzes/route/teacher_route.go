package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	quizcontroller "masjidku_backend/internals/features/school/submissions_assesments/quizzes/controller"
)

/*
Catatan:
- Pasang middleware RequireTeacher di parent router `r` (prefix: /api/t).
- Kita expose 2 varian base segment masjid:
  1) /api/t/:masjid_id/...
  2) /api/t/:masjid_slug/...
- Path hasil:
  - /api/t/:masjid_id/quizzes-teacher/...
  - /api/t/:masjid_id/quiz-questions-teacher/...
  - (alias) /api/t/:masjid_id/quiz-items-teacher/...
  - /api/t/:masjid_id/quizzes-teacher/attempt-answers-teacher/...
  - /api/t/:masjid_id/quizzes-teacher/attempts-teacher/...
  (dan varian yang sama untuk :masjid_slug)
*/

func QuizzesTeacherRoutes(r fiber.Router, db *gorm.DB) {
	// Varian by masjid_id
	mid := r.Group("/:masjid_id")
	mountQuizTeacherRoutes(mid, db)

	// Varian by masjid_slug
	mslug := r.Group("/:masjid_slug")
	mountQuizTeacherRoutes(mslug, db)
}

func mountQuizTeacherRoutes(base fiber.Router, db *gorm.DB) {
	// ============================
	// QUIZZES (master) -> /.../quizzes-teacher
	// ============================
	quizCtrl := quizcontroller.NewQuizController(db)
	quizzes := base.Group("/quizzes-teacher")

	quizzes.Post("/",      quizCtrl.Create)   // POST   /api/t/:masjid_x/quizzes-teacher
	quizzes.Patch("/:id",  quizCtrl.Patch)    // PATCH  /api/t/:masjid_x/quizzes-teacher/:id
	quizzes.Delete("/:id", quizCtrl.Delete)   // DELETE /api/t/:masjid_x/quizzes-teacher/:id

	// ============================
	// QUIZ QUESTIONS (soal & opsi JSONB)
	// -> /.../quiz-questions-teacher  (+ alias /.../quiz-items-teacher)
	// ============================
	qqCtrl := quizcontroller.NewQuizQuestionsController(db)
	qqMain := base.Group("/quiz-questions-teacher")
	mountQuizQuestionsGroup(qqMain, qqCtrl)

	// Alias kompatibel
	qqAlias := base.Group("/quiz-items-teacher")
	mountQuizQuestionsGroup(qqAlias, qqCtrl)

	// ============================
	// USER QUIZ ATTEMPT ANSWERS (guru review/ubah)
	// child dari /quizzes-teacher -> /attempt-answers-teacher
	// ============================
	uqaCtrl := quizcontroller.NewUserQuizAttemptAnswersController(db)
	ans := quizzes.Group("/attempt-answers-teacher")

	ans.Get("/",       uqaCtrl.List)     // GET    /api/t/:masjid_x/quizzes-teacher/attempt-answers-teacher?attempt_id=...&question_id=...
	ans.Post("/",      uqaCtrl.Create)   // POST   /api/t/:masjid_x/quizzes-teacher/attempt-answers-teacher
	ans.Patch("/:id",  uqaCtrl.Patch)    // PATCH  /api/t/:masjid_x/quizzes-teacher/attempt-answers-teacher/:id
	ans.Delete("/:id", uqaCtrl.Delete)   // DELETE /api/t/:masjid_x/quizzes-teacher/attempt-answers-teacher/:id

	// ============================
	// USER QUIZ ATTEMPTS
	// child dari /quizzes-teacher -> /attempts-teacher
	// ============================
	uqAttemptCtrl := quizcontroller.NewUserQuizAttemptsController(db)
	attempts := quizzes.Group("/attempts-teacher")

	attempts.Get("/",      uqAttemptCtrl.List)    // GET    /api/t/:masjid_x/quizzes-teacher/attempts-teacher?quiz_id=&student_id=&status=&active_only=true
	attempts.Post("/",     uqAttemptCtrl.Create)  // POST   /api/t/:masjid_x/quizzes-teacher/attempts-teacher
	attempts.Patch("/:id", uqAttemptCtrl.Patch)   // PATCH  /api/t/:masjid_x/quizzes-teacher/attempts-teacher/:id
	attempts.Delete("/:id", uqAttemptCtrl.Delete) // DELETE /api/t/:masjid_x/quizzes-teacher/attempts-teacher/:id
}

// Hindari duplikasi handler antara quiz-questions-teacher dan alias quiz-items-teacher
func mountQuizQuestionsGroup(g fiber.Router, qqCtrl *quizcontroller.QuizQuestionsController) {
	g.Get("/",      qqCtrl.List)    // GET    /.../quiz-questions-teacher?quiz_id=&type=&q=&page=&per_page=&sort=
	g.Get("/list",  qqCtrl.List)    // alias
	g.Post("/",     qqCtrl.Create)  // POST   /.../quiz-questions-teacher
	g.Patch("/:id", qqCtrl.Patch)   // PATCH  /.../quiz-questions-teacher/:id
	g.Delete("/:id", qqCtrl.Delete) // DELETE /.../quiz-questions-teacher/:id
}
