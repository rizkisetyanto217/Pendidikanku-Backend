// internals/route/classes_admin_routes.go
package route

import (
	classctrl "masjidku_backend/internals/features/school/classes/classes/controller"

	masjidkuMiddleware "masjidku_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func ClassAdminRoutes(admin fiber.Router, db *gorm.DB) {
	// Controller classes
	classHandler := classctrl.NewClassController(db)

	classes := admin.Group("/classes", masjidkuMiddleware.IsMasjidAdmin())
	{
		classes.Post("/", classHandler.CreateClass)
		classes.Get("/list", classHandler.ListClasses)
		classes.Get("/search", classHandler.SearchWithSubjects)
		classes.Get("/slug/:slug", classHandler.GetClassBySlug)
		classes.Get("/:id", classHandler.GetClassByID)
		classes.Put("/:id", classHandler.UpdateClass)
		classes.Delete("/:id", classHandler.SoftDeleteClass)
	}

	// // Controller user classes
	// userClassHandler := classctrl.NewUserClassController(db)

	// userClasses := admin.Group("/user-classes", masjidkuMiddleware.IsMasjidAdmin())
	// {
	// 	// userClasses.Post("/", userClassHandler.CreateUserClass)
	// 	userClasses.Get("/", userClassHandler.ListUserClasses)
	// 	userClasses.Get("/:id", userClassHandler.GetUserClassByID)
	// 	userClasses.Put("/:id", userClassHandler.UpdateUserClass)
	// 	userClasses.Delete("/:id", userClassHandler.EndUserClass)
	// 	userClasses.Delete("/remove/:id", userClassHandler.DeleteUserClass)
	// }

}
