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

	// Controller user classes
	userClassHandler := classctrl.NewUserClassController(db)

	userClasses := admin.Group("/user-classes", masjidkuMiddleware.IsMasjidAdmin())
	{
		// userClasses.Post("/", userClassHandler.CreateUserClass)
		userClasses.Get("/", userClassHandler.ListUserClasses)
		userClasses.Get("/:id", userClassHandler.GetUserClassByID)
		userClasses.Put("/:id", userClassHandler.UpdateUserClass)
		userClasses.Delete("/:id", userClassHandler.EndUserClass)
		userClasses.Delete("/remove/:id", userClassHandler.DeleteUserClass)
	}

	// ===== CPO (Class Pricing Options) - ADMIN DKM =====
	cpoHandler := classctrl.NewCPOController(db)

	// by class_id (dalam konteks kelas)
	classPricing := admin.Group("/classes", masjidkuMiddleware.IsMasjidAdmin())
	{
		cpo := classPricing.Group("/:class_id/pricing-options")
		cpo.Post("/", cpoHandler.AdminCreateCPO)          // create
		cpo.Get("/", cpoHandler.AdminListCPO)             // list (support ?type=&include_deleted=)
		cpo.Get("/latest", cpoHandler.AdminLatestCPO)     // latest per type (support ?type=)
	}

	// by pricing option id (operasi langsung ke item)
	cpoByID := admin.Group("/pricing-options", masjidkuMiddleware.IsMasjidAdmin())
	{
		cpoByID.Get("/:id", cpoHandler.AdminGetCPOByID)
		cpoByID.Put("/:id", cpoHandler.AdminReplaceCPO)   // PUT (full replace)
		cpoByID.Delete("/:id", cpoHandler.AdminSoftDeleteCPO)
		cpoByID.Post("/:id/restore", cpoHandler.AdminRestoreCPO)
	}
}
