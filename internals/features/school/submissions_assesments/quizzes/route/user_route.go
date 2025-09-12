package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	quizcontroller "masjidku_backend/internals/features/school/submissions_assesments/quizzes/controller"
)

/*
Catatan:
- Base (user/public): /api/u
- Controller sisi user WAJIB filter quizzes_is_published = true.
*/

func QuizzesUserRoutes(r fiber.Router, db *gorm.DB) {
	// ============================
	// QUIZZES (read-only publik)
	// ============================
	quizCtrl := quizcontroller.NewQuizController(db)
	quizzes := r.Group("/quizzes") // -> /api/u/quizzes

	quizzes.Get("/list", quizCtrl.List)   // alias

	// ============================
	// QUIZ QUESTIONS (read-only)
	// ============================
	qqCtrl := quizcontroller.NewQuizQuestionsController(db)
	qq := r.Group("/quiz-questions") // -> /api/u/quiz-questions

	qq.Get("/",     qqCtrl.List)    // GET /api/u/quiz-questions?quiz_id=&type=&q=&page=&per_page=&sort=
	qq.Get("/list", qqCtrl.List)    // alias

	// ============================
	// USER QUIZ ATTEMPTS (user)
	// tetap nested di /quizzes
	// ============================
	uqAttemptCtrl := quizcontroller.NewUserQuizAttemptsController(db)
	attempts := r.Group("/user-quiz-attempts") // -> /api/u/quizzes/attempts

	attempts.Get("/list",  uqAttemptCtrl.List) // alias
	attempts.Post("/",     uqAttemptCtrl.Create)
	attempts.Patch("/:id", uqAttemptCtrl.Patch)
	attempts.Delete("/:id", uqAttemptCtrl.Delete)

	// ============================
	// USER QUIZ ATTEMPT ANSWERS (submit jawaban)
	// tetap nested di /quizzes
	// ============================
	uqaCtrl := quizcontroller.NewUserQuizAttemptAnswersController(db)
	ans := r.Group("/user-quiz-attempt-answers") // -> /api/u/quizzes/attempt-answers

	ans.Get("/list",  uqaCtrl.List) // alias
	ans.Post("/",     uqaCtrl.Create)
	// (User tidak memiliki PATCH/DELETE untuk jawaban)
}
