package routes

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	controllers "masjidku_backend/internals/features/school/academics/certificates/controller" // sesuaikan bila path model berbeda
)

func CertificateUserRoutes(r fiber.Router, db *gorm.DB) {
	ussCtl := controllers.NewUserSubjectSummaryController(db)

	// langsung pakai full prefix
	uss := r.Group("/user-subject-summary")
	uss.Get("/", ussCtl.List)     // batasi hasil via middleware (student_id dari token)
}
