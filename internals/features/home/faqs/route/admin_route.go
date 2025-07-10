package route

import (
	"masjidku_backend/internals/features/home/faqs/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func FaqQuestionAdminRoutes(admin fiber.Router, db *gorm.DB) {
	faqQuestionCtrl := controller.NewFaqQuestionController(db)
	faqAnswerCtrl := controller.NewFaqAnswerController(db)

	// Group: /faq-questions
	faqQuestion := admin.Group("/faq-questions")
	faqQuestion.Get("/", faqQuestionCtrl.GetAllFaqQuestions)      // 📄 Semua pertanyaan
	faqQuestion.Get("/:id", faqQuestionCtrl.GetFaqQuestionByID)   // 🔍 Detail pertanyaan
	faqQuestion.Put("/:id", faqQuestionCtrl.UpdateFaqQuestion)    // ✏️ Edit pertanyaan
	faqQuestion.Delete("/:id", faqQuestionCtrl.DeleteFaqQuestion) // ❌ Hapus pertanyaan

	// Group: /faq-answers
	faqAnswer := admin.Group("/faq-answers")
	faqAnswer.Post("/", faqAnswerCtrl.CreateFaqAnswer)      // ➕ Tambah jawaban
	faqAnswer.Get("/:id", faqAnswerCtrl.GetFaqAnswerByID)   // 🔍 Detail jawaban
	faqAnswer.Put("/:id", faqAnswerCtrl.UpdateFaqAnswer)    // ✏️ Edit jawaban
	faqAnswer.Delete("/:id", faqAnswerCtrl.DeleteFaqAnswer) // ❌ Hapus jawaban
}