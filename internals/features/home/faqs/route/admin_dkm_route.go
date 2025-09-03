package route

import (
	"masjidku_backend/internals/constants"
	"masjidku_backend/internals/features/home/faqs/controller"
	authMiddleware "masjidku_backend/internals/middlewares/auth"

	// kalau FAQ per-masjid, aktifkan ini:
	// masjidkuMiddleware "masjidku_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// CUD & manajemen FAQ (login + admin/dkm/owner)
func FaqAdminRoutes(api fiber.Router, db *gorm.DB) {
	admin := api.Group("/",
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola FAQ"),
			constants.AdminAndAbove,
		),
		// kalau FAQ scoped ke masjid, aktifkan guard ini juga:
		// masjidkuMiddleware.IsMasjidAdmin(),
	)

	faqQuestionCtrl := controller.NewFaqQuestionController(db)
	faqAnswerCtrl  := controller.NewFaqAnswerController(db)

	// /faq-questions (admin)
	fq := admin.Group("/faq-questions")
	fq.Get("/",  faqQuestionCtrl.GetAllFaqQuestions)     // list internal/dashboard
	fq.Get("/:id", faqQuestionCtrl.GetFaqQuestionByID)   // detail
	fq.Put("/:id",  faqQuestionCtrl.UpdateFaqQuestion)   // edit
	fq.Delete("/:id", faqQuestionCtrl.DeleteFaqQuestion) // hapus
	// (opsional) kalau admin juga boleh create pertanyaan:
	// fq.Post("/", faqQuestionCtrl.CreateFaqQuestion)

	// /faq-answers (admin)
	fa := admin.Group("/faq-answers")
	fa.Post("/",   faqAnswerCtrl.CreateFaqAnswer)     // create jawaban
	fa.Get("/:id", faqAnswerCtrl.GetFaqAnswerByID)    // detail jawaban
	fa.Put("/:id", faqAnswerCtrl.UpdateFaqAnswer)     // edit jawaban
	fa.Delete("/:id", faqAnswerCtrl.DeleteFaqAnswer)  // hapus jawaban
}
