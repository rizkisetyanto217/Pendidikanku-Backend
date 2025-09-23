// file: internals/features/system/holidays/routes/admin_routes.go
package routes

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	nhctl "masjidku_backend/internals/features/school/academics/schedules/controller"
	helperAuth "masjidku_backend/internals/helpers/auth"
)

// NationalHolidayAdminRoutes: owner-only CRUD (POST/PATCH/DELETE)
func NationalHolidayAdminRoutes(admin fiber.Router, db *gorm.DB) {
	ctl := nhctl.NewNationHoliday(db, validator.New())

	grp := admin.Group("/holidays/national", helperAuth.OwnerOnly())

	grp.Post("/", ctl.Create)
	grp.Patch("/:id", ctl.Patch)
	grp.Delete("/:id", ctl.Delete)
}
