package route

import (
	"schoolku_backend/internals/constants"
	"schoolku_backend/internals/features/home/faqs/controller"
	authMiddleware "schoolku_backend/internals/middlewares/auth"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Pengguna login bisa kirim pertanyaan & baca FAQ
func FaqUserRoutes(api fiber.Router, db *gorm.DB) {
	user := api.Group("/",
		authMiddleware.OnlyRolesSlice(
			"❌ Hanya pengguna terautentikasi yang boleh mengakses fitur FAQ.",
			constants.AllowedRoles,
		),
	)

	faqQuestionCtrl := controller.NewFaqQuestionController(db)
	faqAnswerCtrl := controller.NewFaqAnswerController(db)

	// /faq-questions (user)
	fq := user.Group("/faq-questions")
	fq.Post("/", faqQuestionCtrl.CreateFaqQuestion)    // user kirim pertanyaan
	fq.Get("/", faqQuestionCtrl.GetAllFaqQuestions)    // list (bisa filter by user di controller)
	fq.Get("/:id", faqQuestionCtrl.GetFaqQuestionByID) // detail pertanyaan

	// /faq-answers (user)
	fa := user.Group("/faq-answers")
	fa.Get("/:id", faqAnswerCtrl.GetFaqAnswerByID) // lihat jawaban
}
