package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	quizcontroller "schoolku_backend/internals/features/school/submissions_assesments/quizzes/controller"
)

/*
Catatan:
- Parent router sudah di-mount dengan prefix /api/a dan middleware RequireAdmin.
- School context diambil dari helper ResolveSchoolContext (token / header / dsb),
  bukan dari path (:school_id / :school_slug).
*/

func QuizzesAdminRoutes(r fiber.Router, db *gorm.DB) {
	// Base: /api/a/quizzes
	g := r.Group("/quizzes")
	mountQuizRoutes(g, db)
}

// mountQuizRoutes mendaftarkan semua endpoint di bawah group yg diberikan
func mountQuizRoutes(g fiber.Router, db *gorm.DB) {
	// QUIZZES (master)
	ctrl := quizcontroller.NewQuizController(db)

	// List: sediakan "/" dan "/list" sebagai alias
	g.Get("/", ctrl.List)         // GET /api/a/quizzes
	g.Get("/list", ctrl.List)     // GET /api/a/quizzes/list
	g.Post("/", ctrl.Create)      // POST /api/a/quizzes
	g.Patch("/:id", ctrl.Patch)   // PATCH /api/a/quizzes/:id
	g.Delete("/:id", ctrl.Delete) // DELETE /api/a/quizzes/:id

	// QUIZ QUESTIONS (soal & opsi dalam satu baris)
	qqCtrl := quizcontroller.NewQuizQuestionsController(db)
	qs := g.Group("/questions") // -> /api/a/quizzes/questions

	qs.Get("/", qqCtrl.List)         // GET /api/a/quizzes/questions?quiz_id=&type=&q=&page=&per_page=&sort=
	qs.Get("/list", qqCtrl.List)     // alias
	qs.Post("/", qqCtrl.Create)      // POST /api/a/quizzes/questions
	qs.Patch("/:id", qqCtrl.Patch)   // PATCH /api/a/quizzes/questions/:id
	qs.Delete("/:id", qqCtrl.Delete) // DELETE /api/a/quizzes/questions/:id

	// USER QUIZ ATTEMPT ANSWERS
	uqaCtrl := quizcontroller.NewStudentQuizAttemptAnswersController(db)
	ans := g.Group("/attempt-answers") // -> /api/a/quizzes/attempt-answers

	ans.Get("/", uqaCtrl.List)         // GET    /api/a/quizzes/attempt-answers?attempt_id=...&question_id=...
	ans.Post("/", uqaCtrl.Create)      // POST   /api/a/quizzes/attempt-answers
	ans.Patch("/:id", uqaCtrl.Patch)   // PATCH  /api/a/quizzes/attempt-answers/:id
	ans.Delete("/:id", uqaCtrl.Delete) // DELETE /api/a/quizzes/attempt-answers/:id

	// USER QUIZ ATTEMPTS
	uqAttemptCtrl := quizcontroller.NewStudentQuizAttemptsController(db)
	attempts := g.Group("/attempts") // -> /api/a/quizzes/attempts

	attempts.Get("/", uqAttemptCtrl.List)         // GET    /api/a/quizzes/attempts?quiz_id=&student_id=&status=&active_only=true
	attempts.Post("/", uqAttemptCtrl.Create)      // POST   /api/a/quizzes/attempts
	attempts.Patch("/:id", uqAttemptCtrl.Patch)   // PATCH  /api/a/quizzes/attempts/:id
	attempts.Delete("/:id", uqAttemptCtrl.Delete) // DELETE /api/a/quizzes/attempts/:id
}
