package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	quizcontroller "masjidku_backend/internals/features/school/submissions_assesments/quizzes/controller"
)

/*
Catatan:
- Pasang middleware auth sesuai kebutuhan pada router parent:
  - AdminRoutes: RequireAdmin
  - TeacherRoutes: RequireTeacher
  - UserRoutes: optional/none (read-only publik)
*/

// =========================================
// Admin Routes (CRUD penuh)
// Base: /api/a/quizzes
// =========================================
func QuizzesAdminRoutes(r fiber.Router, db *gorm.DB) {
	// Controller untuk master quiz
	ctrl := quizcontroller.NewQuizController(db)

	g := r.Group("/quizzes")
	g.Get("/list", ctrl.List)     // GET   /quizzes/list
	g.Get("/:id", ctrl.GetByID)   // GET   /quizzes/:id
	g.Post("/", ctrl.Create)      // POST  /quizzes
	g.Patch("/:id", ctrl.Patch)   // PATCH /quizzes/:id
	g.Delete("/:id", ctrl.Delete) // DELETE /quizzes/:id

	// ============================
	// QUIZ ITEMS (soal & opsi)
	// Base: /api/a/quizzes/items
	// ============================
	itemsCtrl := quizcontroller.NewQuizItemsController(db)
	items := g.Group("/items")

	items.Post("/", itemsCtrl.Create)                                  // POST   /quizzes/items
	items.Post("/bulk-single", itemsCtrl.BulkCreateSingle)             // POST   /quizzes/items/bulk-single
	items.Get("/", itemsCtrl.ListByQuiz)                               // GET    /quizzes/items?quiz_id=...
	items.Get("/by-question/:question_id", itemsCtrl.ListByQuestion)   // GET    /quizzes/items/by-question/:question_id
	items.Get("/:id", itemsCtrl.GetByID)                               // GET    /quizzes/items/:id
	items.Put("/:id", itemsCtrl.Patch)                                 // PUT    /quizzes/items/:id
	items.Delete("/:id", itemsCtrl.Delete)                             // DELETE /quizzes/items/:id

	// =========================================
	// USER QUIZ ATTEMPT ANSWERS
	// Base: /api/a/quizzes/attempt-answers
	// =========================================
	uqaCtrl := quizcontroller.NewUserQuizAttemptAnswersController(db)
	ans := g.Group("/attempt-answers")

	ans.Get("/", uqaCtrl.List)          // GET    /quizzes/attempt-answers?attempt_id=...&question_id=...
	ans.Get("/:id", uqaCtrl.GetByID)    // GET    /quizzes/attempt-answers/:id
	ans.Post("/", uqaCtrl.Create)       // POST   /quizzes/attempt-answers
	ans.Patch("/:id", uqaCtrl.Patch)    // PATCH  /quizzes/attempt-answers/:id
	ans.Delete("/:id", uqaCtrl.Delete)  // DELETE /quizzes/attempt-answers/:id

	// =========================================
	// USER QUIZ ATTEMPTS
	// Base: /api/a/quizzes/attempts
	// =========================================
	uqAttemptCtrl := quizcontroller.NewUserQuizAttemptsController(db)
	attempts := g.Group("/attempts")

	attempts.Get("/", uqAttemptCtrl.List)         // GET    /quizzes/attempts?quiz_id=&student_id=&status=&active_only=true
	attempts.Get("/:id", uqAttemptCtrl.GetByID)   // GET    /quizzes/attempts/:id
	attempts.Post("/", uqAttemptCtrl.Create)      // POST   /quizzes/attempts
	attempts.Patch("/:id", uqAttemptCtrl.Patch)   // PATCH  /quizzes/attempts/:id
	attempts.Delete("/:id", uqAttemptCtrl.Delete) // DELETE /quizzes/attempts/:id
}
