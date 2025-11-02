package route

import (
	"schoolku_backend/internals/features/home/faqs/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Publik bisa membaca FAQ (read-only)
func AllFaqRoutes(api fiber.Router, db *gorm.DB) {
	faqQuestionCtrl := controller.NewFaqQuestionController(db)
	faqAnswerCtrl := controller.NewFaqAnswerController(db)

	// /faq-questions (public read)
	fq := api.Group("/faq-questions")
	fq.Get("/", faqQuestionCtrl.GetAllFaqQuestions)    // daftar pertanyaan (pastikan controller hanya expose yg layak tampil)
	fq.Get("/:id", faqQuestionCtrl.GetFaqQuestionByID) // detail

	// /faq-answers (public read)
	fa := api.Group("/faq-answers")
	fa.Get("/:id", faqAnswerCtrl.GetFaqAnswerByID) // jawaban
}
