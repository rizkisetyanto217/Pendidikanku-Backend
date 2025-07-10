package route

import (
	"masjidku_backend/internals/features/home/faqs/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AllFaqQuestionRoutes(user fiber.Router, db *gorm.DB) {
	faqQuestionCtrl := controller.NewFaqQuestionController(db)
	faqAnswerCtrl := controller.NewFaqAnswerController(db)

	// Group: /faq-questions
	faqQuestion := user.Group("/faq-questions")
	faqQuestion.Post("/", faqQuestionCtrl.CreateFaqQuestion)    // ➕ Kirim pertanyaan
	faqQuestion.Get("/", faqQuestionCtrl.GetAllFaqQuestions)    // 📄 Semua pertanyaan (bisa difilter per user ID)
	faqQuestion.Get("/:id", faqQuestionCtrl.GetFaqQuestionByID) // 🔍 Detail pertanyaan

	// Group: /faq-answers
	faqAnswer := user.Group("/faq-answers")
	faqAnswer.Get("/:id", faqAnswerCtrl.GetFaqAnswerByID) // 🔍 Jawaban dari pertanyaan tertentu
}