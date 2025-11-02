// file: internals/route/details/without_school_routes.go
package details

import (
	ucsctrl "schoolku_backend/internals/features/school/classes/class_sections/controller" // <-- controller, bukan route

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func ClassSectionUserGlobalRoutes(private fiber.Router, db *gorm.DB) {
	ucsH := ucsctrl.NewStudentClassSectionController(db) // constructor ada di controller
	grp := private.Group("/student-class-sections")
	grp.Post("/join", ucsH.JoinByCodeAutoSchool) // handler global (tanpa :school_id)
}
