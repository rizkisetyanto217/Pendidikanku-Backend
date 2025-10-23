package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	quizcontroller "masjidku_backend/internals/features/school/submissions_assesments/quizzes/controller"
)

/*
Catatan:
- Base (user/public): /api/u
- Controller sisi user WAJIB filter quizzes_is_published = true. (Di-controller, bukan di route)
- Kita expose 2 varian base segment masjid:
  1) /api/u/:masjid_id/...
  2) /api/u/:masjid_slug/...
- Path utama (baru):
  - /api/u/:masjid_x/quizzes/...
  - /api/u/:masjid_x/quiz-questions/...
  - /api/u/:masjid_x/quizzes/attempts/...
  - /api/u/:masjid_x/quizzes/attempt-answers/...
- Alias kompatibel (tetap hidup):
  - /api/u/:masjid_x/user-quiz-attempts/...
  - /api/u/:masjid_x/user-quiz-attempt-answers/...
*/

func QuizzesUserRoutes(r fiber.Router, db *gorm.DB) {
	// Varian by masjid_id
	mid := r.Group("/:masjid_id")
	mountQuizUserRoutes(mid, db)

	// Varian by masjid_slug
	mslug := r.Group("/:masjid_slug")
	mountQuizUserRoutes(mslug, db)
}

func mountQuizUserRoutes(base fiber.Router, db *gorm.DB) {
	// ============================
	// QUIZZES (read-only publik)
	// ============================
	quizCtrl := quizcontroller.NewQuizController(db)
	quizzes := base.Group("/quizzes") // -> /api/u/:masjid_x/quizzes

	quizzes.Get("/",     quizCtrl.List) // GET /api/u/:masjid_x/quizzes
	quizzes.Get("/list", quizCtrl.List) // alias

	// ============================
	// QUIZ QUESTIONS (read-only)
	// ============================
	qqCtrl := quizcontroller.NewQuizQuestionsController(db)
	qq := base.Group("/quiz-questions") // -> /api/u/:masjid_x/quiz-questions

	qq.Get("/",     qqCtrl.List) // GET /api/u/:masjid_x/quiz-questions?quiz_id=&type=&q=&page=&per_page=&sort=
	qq.Get("/list", qqCtrl.List) // alias

	// ============================
	// USER QUIZ ATTEMPTS (user)
	// nested di /quizzes + alias kompatibel
	// ============================
	uqAttemptCtrl := quizcontroller.NewStudentQuizAttemptsController(db)

	// Nested utama
	attempts := quizzes.Group("/attempts") // -> /api/u/:masjid_x/quizzes/attempts
	mountUserAttemptsGroup(attempts, uqAttemptCtrl)

	// Alias kompatibel (rute lama)
	attemptsAlias := base.Group("/user-quiz-attempts") // -> /api/u/:masjid_x/user-quiz-attempts
	mountUserAttemptsGroup(attemptsAlias, uqAttemptCtrl)

	// ============================
	// USER QUIZ ATTEMPT ANSWERS (submit jawaban)
	// nested di /quizzes + alias kompatibel
	// ============================
	uqaCtrl := quizcontroller.NewStudentQuizAttemptAnswersController(db)

	// Nested utama
	ans := quizzes.Group("/attempt-answers") // -> /api/u/:masjid_x/quizzes/attempt-answers
	mountUserAttemptAnswersGroup(ans, uqaCtrl, true /*user mode: no patch/delete*/)

	// Alias kompatibel (rute lama)
	ansAlias := base.Group("/user-quiz-attempt-answers") // -> /api/u/:masjid_x/user-quiz-attempt-answers
	mountUserAttemptAnswersGroup(ansAlias, uqaCtrl, true)
}

// Hindari duplikasi handler untuk attempts (nested & alias)
func mountUserAttemptsGroup(g fiber.Router, ctrl *quizcontroller.StudentQuizAttemptsController) {
	g.Get("/",      ctrl.List)   // GET list (boleh juga dipanggil sebagai /list lewat mapping reverse proxy kalau mau)
	g.Get("/list",  ctrl.List)   // alias
	g.Post("/",     ctrl.Create) // POST create attempt
	g.Patch("/:id", ctrl.Patch)  // PATCH attempt by id
	g.Delete("/:id", ctrl.Delete) // DELETE attempt by id
}

// Hindari duplikasi handler untuk attempt-answers (nested & alias)
func mountUserAttemptAnswersGroup(g fiber.Router, ctrl *quizcontroller.StudentQuizAttemptAnswersController, userMode bool) {
	g.Get("/",     ctrl.List)   // GET list
	g.Get("/list", ctrl.List)   // alias
	g.Post("/",    ctrl.Create) // POST submit jawaban

	// Di user mode, tidak expose PATCH/DELETE (sesuai catatan)
	if !userMode {
		g.Patch("/:id",  ctrl.Patch)
		g.Delete("/:id", ctrl.Delete)
	}
}