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
		classes.Patch("/:id", classHandler.PatchClass)
		classes.Delete("/:id", classHandler.SoftDeleteClass)
	}

	// Controller class parents
	parentHandler := classctrl.NewClassParentController(db, nil)

	classParents := admin.Group("/class-parents", masjidkuMiddleware.IsMasjidAdmin())
	{
		classParents.Post("/", parentHandler.Create)
		classParents.Get("/list", parentHandler.List)
		classParents.Get("/:id", parentHandler.GetByID)
		classParents.Put("/:id", parentHandler.Update)
		classParents.Patch("/:id", parentHandler.Update)
		classParents.Delete("/:id", parentHandler.Delete)
	}

	// Controller user classes
	userClassHandler := classctrl.NewUserClassController(db)
	userClasses := admin.Group("/user-classes", masjidkuMiddleware.IsMasjidAdmin())
	{
		userClasses.Get("/list", userClassHandler.ListUserClasses)
		userClasses.Get("/:id", userClassHandler.GetUserClassByID)
		userClasses.Put("/:id", userClassHandler.UpdateUserClass)
		// userClasses.Delete("/:id", userClassHandler.EndUserClass)
		userClasses.Delete("/remove/:id", userClassHandler.DeleteUserClass)
	}

	// ===== Membership routes (BARU) =====
	membershipHandler := classctrl.NewMembershipController(db, nil)
	m := admin.Group("/membership", masjidkuMiddleware.IsMasjidAdmin())
	{
		// enrollment hooks
		m.Post("/enrollment/activate", membershipHandler.ActivateEnrollment)
		m.Post("/enrollment/deactivate", membershipHandler.DeactivateEnrollment)

		// roles management
		m.Post("/roles/grant", membershipHandler.GrantRole)
		m.Post("/roles/revoke", membershipHandler.RevokeRole)

		// masjid_students ensure
		m.Post("/masjid-students/ensure", membershipHandler.EnsureMasjidStudent)
	}
}
