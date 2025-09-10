package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	quizcontroller "masjidku_backend/internals/features/school/submissions_assesments/quizzes/controller"
)

/*
Catatan:
- Pasang middleware auth pada router parent (RequireTeacher).
- Semua endpoint berada di bawah base: /api/t/quizzes
*/

func QuizzesTeacherRoutes(r fiber.Router, db *gorm.DB) {
	// ============================
	// QUIZZES (master)
	// ============================
	ctrl := quizcontroller.NewQuizController(db)
	g := r.Group("/quizzes")

	g.Get("/list", ctrl.List)   // GET    /api/t/quizzes/list
	g.Get("/:id", ctrl.GetByID) // GET    /api/t/quizzes/:id
	g.Post("/", ctrl.Create)    // POST   /api/t/quizzes
	g.Patch("/:id", ctrl.Patch) // PATCH  /api/t/quizzes/:id
	g.Delete("/:id", ctrl.Delete) // DELETE /api/t/quizzes/:id

	// ============================
	// QUIZ ITEMS (soal & opsi)
	// Base: /api/t/quizzes/items
	// ============================
	itemsCtrl := quizcontroller.NewQuizItemsController(db)
	items := g.Group("/items")

	items.Post("/", itemsCtrl.Create)                                  // POST   /api/t/quizzes/items
	items.Post("/bulk-single", itemsCtrl.BulkCreateSingle)             // POST   /api/t/quizzes/items/bulk-single
	items.Get("/", itemsCtrl.ListByQuiz)                               // GET    /api/t/quizzes/items?quiz_id=...
	items.Get("/by-question/:question_id", itemsCtrl.ListByQuestion)   // GET    /api/t/quizzes/items/by-question/:question_id
	items.Get("/:id", itemsCtrl.GetByID)                               // GET    /api/t/quizzes/items/:id
	items.Put("/:id", itemsCtrl.Patch)                                 // PUT    /api/t/quizzes/items/:id
	items.Delete("/:id", itemsCtrl.Delete)                             // DELETE /api/t/quizzes/items/:id

	// =========================================
	// USER QUIZ ATTEMPT ANSWERS
	// Base: /api/t/quizzes/attempt-answers
	// =========================================
	uqaCtrl := quizcontroller.NewUserQuizAttemptAnswersController(db)
	ans := g.Group("/attempt-answers")

	ans.Get("/", uqaCtrl.List)          // GET    /api/t/quizzes/attempt-answers?attempt_id=...&question_id=...
	ans.Get("/:id", uqaCtrl.GetByID)    // GET    /api/t/quizzes/attempt-answers/:id
	ans.Post("/", uqaCtrl.Create)       // POST   /api/t/quizzes/attempt-answers
	ans.Patch("/:id", uqaCtrl.Patch)    // PATCH  /api/t/quizzes/attempt-answers/:id
	ans.Delete("/:id", uqaCtrl.Delete)  // DELETE /api/t/quizzes/attempt-answers/:id

	// =========================================
	// USER QUIZ ATTEMPTS
	// Base: /api/t/quizzes/attempts
	// =========================================
	uqAttemptCtrl := quizcontroller.NewUserQuizAttemptsController(db)
	attempts := g.Group("/attempts")

	attempts.Get("/", uqAttemptCtrl.List)         // GET    /api/t/quizzes/attempts?quiz_id=&student_id=&status=&active_only=true
	attempts.Get("/:id", uqAttemptCtrl.GetByID)   // GET    /api/t/quizzes/attempts/:id
	attempts.Post("/", uqAttemptCtrl.Create)      // POST   /api/t/quizzes/attempts
	attempts.Patch("/:id", uqAttemptCtrl.Patch)   // PATCH  /api/t/quizzes/attempts/:id
	attempts.Delete("/:id", uqAttemptCtrl.Delete) // DELETE /api/t/quizzes/attempts/:id
}
