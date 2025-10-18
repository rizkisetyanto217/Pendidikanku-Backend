// file: internals/route/details/without_masjid_routes.go
package details

import (
	ucsctrl "masjidku_backend/internals/features/school/classes/class_sections/controller" // <-- controller, bukan route

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func ClassSectionUserGlobalRoutes(private fiber.Router, db *gorm.DB) {
	ucsH := ucsctrl.NewUserClassSectionController(db) // constructor ada di controller
	grp := private.Group("/user-class-sections")
	grp.Post("/join", ucsH.JoinByCodeAutoMasjid) // handler global (tanpa :masjid_id)
}
