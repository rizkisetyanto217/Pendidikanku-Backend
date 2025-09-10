package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	quizcontroller "masjidku_backend/internals/features/school/submissions_assesments/quizzes/controller"
)

/*
Catatan:
- Mount parent router dengan prefix /api/a dan middleware RequireAdmin.
- Base group di sini: /api/a/quizzes
*/

func QuizzesAdminRoutes(r fiber.Router, db *gorm.DB) {
	// ============================
	// QUIZZES (master)
	// ============================
	ctrl := quizcontroller.NewQuizController(db)
	g := r.Group("/quizzes") // -> /api/a/quizzes

	// List: sediakan "/" dan "/list" sebagai alias
	g.Get("/",     ctrl.List)    // GET /api/a/quizzes
	g.Get("/list", ctrl.List)    // GET /api/a/quizzes/list
	g.Post("/",    ctrl.Create)  // POST /api/a/quizzes
	g.Patch("/:id", ctrl.Patch)  // PATCH /api/a/quizzes/:id
	g.Delete("/:id", ctrl.Delete) // DELETE /api/a/quizzes/:id

	// ============================
	// QUIZ QUESTIONS (soal & opsi dalam satu baris)
	// Base: /api/a/quizzes/questions
	// ============================
	qqCtrl := quizcontroller.NewQuizQuestionsController(db)
	qs := g.Group("/questions") // -> /api/a/quizzes/questions

	// List: sediakan "/" dan "/list" sebagai alias
	qs.Get("/",     qqCtrl.List)    // GET /api/a/quizzes/questions?quiz_id=&type=&q=&page=&per_page=&sort=
	qs.Get("/list", qqCtrl.List)    // alias
	qs.Get("/:id",  qqCtrl.GetByID) // GET /api/a/quizzes/questions/:id
	qs.Post("/",    qqCtrl.Create)  // POST /api/a/quizzes/questions
	qs.Patch("/:id", qqCtrl.Patch)  // PATCH /api/a/quizzes/questions/:id
	qs.Delete("/:id", qqCtrl.Delete) // DELETE /api/a/quizzes/questions/:id

	// ============================
	// USER QUIZ ATTEMPT ANSWERS
	// Base: /api/a/quizzes/attempt-answers
	// ============================
	uqaCtrl := quizcontroller.NewUserQuizAttemptAnswersController(db)
	ans := g.Group("/attempt-answers") // -> /api/a/quizzes/attempt-answers

	ans.Get("/",     uqaCtrl.List)     // GET    /api/a/quizzes/attempt-answers?attempt_id=...&question_id=...
	ans.Get("/:id",  uqaCtrl.GetByID)  // GET    /api/a/quizzes/attempt-answers/:id
	ans.Post("/",    uqaCtrl.Create)   // POST   /api/a/quizzes/attempt-answers
	ans.Patch("/:id", uqaCtrl.Patch)   // PATCH  /api/a/quizzes/attempt-answers/:id
	ans.Delete("/:id", uqaCtrl.Delete) // DELETE /api/a/quizzes/attempt-answers/:id

	// ============================
	// USER QUIZ ATTEMPTS
	// Base: /api/a/quizzes/attempts
	// ============================
	uqAttemptCtrl := quizcontroller.NewUserQuizAttemptsController(db)
	attempts := g.Group("/attempts") // -> /api/a/quizzes/attempts

	attempts.Get("/",     uqAttemptCtrl.List)    // GET    /api/a/quizzes/attempts?quiz_id=&student_id=&status=&active_only=true
	attempts.Post("/",    uqAttemptCtrl.Create)  // POST   /api/a/quizzes/attempts
	attempts.Patch("/:id", uqAttemptCtrl.Patch)  // PATCH  /api/a/quizzes/attempts/:id
	attempts.Delete("/:id", uqAttemptCtrl.Delete) // DELETE /api/a/quizzes/attempts/:id
}
