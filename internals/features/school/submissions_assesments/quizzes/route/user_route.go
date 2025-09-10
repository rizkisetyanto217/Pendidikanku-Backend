package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	quizcontroller "masjidku_backend/internals/features/school/submissions_assesments/quizzes/controller"
)

/*
Catatan:
- UserRoutes (publik read-only + submit jawaban)
- Base: /api/u/quizzes
- Filter di controller wajib pastikan hanya quizzes_is_published = true
*/

// =========================================
// User Routes (Publik Read-Only + Create Attempt Answer)
// Base: /api/u/quizzes
// =========================================
func QuizzesUserRoutes(r fiber.Router, db *gorm.DB) {
	ctrl := quizcontroller.NewQuizController(db)
	g := r.Group("/quizzes")

	// Quizzes
	g.Get("/list", ctrl.List)   // GET /api/u/quizzes/list
	g.Get("/:id", ctrl.GetByID) // GET /api/u/quizzes/:id

	// ============================
	// QUIZ ITEMS (soal & opsi) — read only
	// Base: /api/u/quizzes/items
	// ============================
	itemsCtrl := quizcontroller.NewQuizItemsController(db)
	items := g.Group("/items")

	items.Get("/", itemsCtrl.ListByQuiz)                             // GET /api/u/quizzes/items?quiz_id=...
	items.Get("/by-question/:question_id", itemsCtrl.ListByQuestion) // GET /api/u/quizzes/items/by-question/:question_id
	items.Get("/:id", itemsCtrl.GetByID)                             // GET /api/u/quizzes/items/:id

		// =========================================
	// USER QUIZ ATTEMPTS
	// Base: /api/a/quizzes/attempts
	// =========================================
	uqAttemptCtrl := quizcontroller.NewUserQuizAttemptsController(db)
	attempts := g.Group("/attempts")

	attempts.Get("/", uqAttemptCtrl.List)        // GET    /api/a/quizzes/attempts?quiz_id=&student_id=&status=&active_only=true
	attempts.Get("/:id", uqAttemptCtrl.GetByID)  // GET    /api/a/quizzes/attempts/:id
	attempts.Post("/", uqAttemptCtrl.Create)     // POST   /api/a/quizzes/attempts
	attempts.Patch("/:id", uqAttemptCtrl.Patch)  // PATCH  /api/a/quizzes/attempts/:id
	attempts.Delete("/:id", uqAttemptCtrl.Delete) // DELETE /api/a/quizzes/attempts/:id

	// =========================================
	// USER QUIZ ATTEMPT ANSWERS
	// Base: /api/u/quizzes/attempt-answers
	// =========================================
	uqaCtrl := quizcontroller.NewUserQuizAttemptAnswersController(db)
	ans := g.Group("/attempt-answers")

	ans.Get("/", uqaCtrl.List)       // GET  /api/u/quizzes/attempt-answers?attempt_id=...
	ans.Get("/:id", uqaCtrl.GetByID) // GET  /api/u/quizzes/attempt-answers/:id
	ans.Post("/", uqaCtrl.Create)    // POST /api/u/quizzes/attempt-answers
	// ⚠️ Tidak ada PATCH/DELETE untuk user
}
