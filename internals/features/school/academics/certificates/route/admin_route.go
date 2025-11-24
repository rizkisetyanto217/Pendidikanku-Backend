package routes

// import (
// 	"github.com/gofiber/fiber/v2"
// 	"gorm.io/gorm"

// 	controllers "madinahsalam_backend/internals/features/school/academics/certificates/controller" // sesuaikan bila path model berbeda
// )

// func CertificateAdminRoutes(r fiber.Router, db *gorm.DB) {
// 	ussCtl := controllers.NewUserSubjectSummaryController(db)

// 	// langsung pakai full prefix
// 	uss := r.Group("/user-subject-summary")
// 	uss.Get("/", ussCtl.List)
// 	uss.Post("/", ussCtl.Create)
// 	uss.Patch("/:id", ussCtl.Update)
// 	uss.Delete("/:id", ussCtl.Delete)
// 	uss.Post("/:id/restore", ussCtl.Restore)
// }
