package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	quizcontroller "masjidku_backend/internals/features/school/submissions_assesments/quizzes/controller"
)

/*
Catatan:
- Parent router sudah di-mount dengan prefix /api/a dan middleware RequireAdmin.
- Kita expose 2 varian base group:
  1) /api/a/:masjid_id/quizzes     (by UUID)
  2) /api/a/:masjid_slug/quizzes   (by slug)
  Keduanya dibaca oleh helper ResolveMasjidContext (path → header → cookie → query → host → token)
*/

func QuizzesAdminRoutes(r fiber.Router, db *gorm.DB) {
	// ============================
	// QUIZZES (master)
	// ============================
	// Varian by masjid_id
	mid := r.Group("/:masjid_id/quizzes") // -> /api/a/:masjid_id/quizzes
	mountQuizRoutes(mid, db)

	// Varian by masjid_slug
	mslug := r.Group("/:masjid_slug/quizzes") // -> /api/a/:masjid_slug/quizzes
	mountQuizRoutes(mslug, db)
}

// mountQuizRoutes mendaftarkan semua endpoint di bawah group yg diberikan
func mountQuizRoutes(g fiber.Router, db *gorm.DB) {
	// QUIZZES (master)
	ctrl := quizcontroller.NewQuizController(db)

	// List: sediakan "/" dan "/list" sebagai alias
	g.Get("/",      ctrl.List)    // GET /.../quizzes
	g.Get("/list",  ctrl.List)    // GET /.../quizzes/list
	g.Post("/",     ctrl.Create)  // POST /.../quizzes
	g.Patch("/:id", ctrl.Patch)   // PATCH /.../quizzes/:id
	g.Delete("/:id", ctrl.Delete) // DELETE /.../quizzes/:id

	// QUIZ QUESTIONS (soal & opsi dalam satu baris)
	qqCtrl := quizcontroller.NewQuizQuestionsController(db)
	qs := g.Group("/questions") // -> /.../quizzes/questions

	qs.Get("/",      qqCtrl.List)   // GET /.../quizzes/questions?quiz_id=&type=&q=&page=&per_page=&sort=
	qs.Get("/list",  qqCtrl.List)   // alias
	qs.Post("/",     qqCtrl.Create) // POST /.../quizzes/questions
	qs.Patch("/:id", qqCtrl.Patch)  // PATCH /.../quizzes/questions/:id
	qs.Delete("/:id", qqCtrl.Delete) // DELETE /.../quizzes/questions/:id

	// USER QUIZ ATTEMPT ANSWERS
	uqaCtrl := quizcontroller.NewUserQuizAttemptAnswersController(db)
	ans := g.Group("/attempt-answers") // -> /.../quizzes/attempt-answers

	ans.Get("/",      uqaCtrl.List)    // GET    /.../quizzes/attempt-answers?attempt_id=...&question_id=...
	ans.Post("/",     uqaCtrl.Create)  // POST   /.../quizzes/attempt-answers
	ans.Patch("/:id", uqaCtrl.Patch)   // PATCH  /.../quizzes/attempt-answers/:id
	ans.Delete("/:id", uqaCtrl.Delete) // DELETE /.../quizzes/attempt-answers/:id

	// USER QUIZ ATTEMPTS
	uqAttemptCtrl := quizcontroller.NewUserQuizAttemptsController(db)
	attempts := g.Group("/attempts") // -> /.../quizzes/attempts

	attempts.Get("/",      uqAttemptCtrl.List)   // GET    /.../quizzes/attempts?quiz_id=&student_id=&status=&active_only=true
	attempts.Post("/",     uqAttemptCtrl.Create) // POST   /.../quizzes/attempts
	attempts.Patch("/:id", uqAttemptCtrl.Patch)  // PATCH  /.../quizzes/attempts/:id
	attempts.Delete("/:id", uqAttemptCtrl.Delete) // DELETE /.../quizzes/attempts/:id
}
