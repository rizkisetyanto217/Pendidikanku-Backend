// internals/features/lembaga/user_quran_records/routes/user_quran_record_routes.go
package routes

import (
	uqCtl "masjidku_backend/internals/features/school/class_attendance_result/user_result/user_quran/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

//   - /admin/user-quran-urls/...
func UserQuranAdminRoutes(r fiber.Router, db *gorm.DB) {
	// controllers
	rec := uqCtl.NewUserQuranRecordController(db)
	uurl := uqCtl.NewUserQuranURLController(db)

	// ----- RECORDS -----
	recGrp := r.Group("/user-quran-records")
	recGrp.Get("/", rec.List)
	recGrp.Post("/", rec.Create)
	recGrp.Get("/:id", rec.GetByID)
	recGrp.Patch("/:id", rec.Update)
	recGrp.Delete("/:id", rec.Delete)

	// ----- URLS (child of records) -----
	urlGrp := r.Group("/user-quran-urls")
	urlGrp.Get("/", uurl.List)
	urlGrp.Get("/:id", uurl.GetByID)
	urlGrp.Post("/", uurl.Create)         // create by JSON (href already provided)
	urlGrp.Post("/upload", uurl.Upload)   // multipart upload to OSS
	urlGrp.Patch("/:id", uurl.Update)
	urlGrp.Delete("/:id", uurl.Delete)
	urlGrp.Delete("/:id/oss", uurl.DeleteFromOSS) // optional hard delete on OSS
}