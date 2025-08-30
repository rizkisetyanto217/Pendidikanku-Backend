// internals/features/lembaga/user_quran_records/routes/user_quran_record_routes.go
package routes

import (
	uqCtl "masjidku_backend/internals/features/school/class_attendance_result/user_result/user_quran/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func UserQuranUserRoutes(r fiber.Router, db *gorm.DB) {
	// controllers
	rec := uqCtl.NewUserQuranRecordController(db)
	uurl := uqCtl.NewUserQuranURLController(db)

	// ----- RECORDS -----
	recGrp := r.Group("/user-quran-records")
	recGrp.Get("/", rec.List)
	recGrp.Get("/:id", rec.GetByID)

	// ----- URLS -----
	urlGrp := r.Group("/user-quran-urls")
	urlGrp.Get("/", uurl.List)
	urlGrp.Get("/:id", uurl.GetByID)
	urlGrp.Post("/upload", uurl.Upload) // izinkan user upload jika diperlukan
}