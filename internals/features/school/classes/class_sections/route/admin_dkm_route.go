// internals/features/lembaga/classes/sections/main/route/admin_route.go
package route

import (
	"schoolku_backend/internals/constants"
	sectionctrl "schoolku_backend/internals/features/school/classes/class_sections/controller"
	authMiddleware "schoolku_backend/internals/middlewares/auth"
	schoolkuMiddleware "schoolku_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func ClassSectionAdminRoutes(api fiber.Router, db *gorm.DB) {
	// Controllers
	sectionH := sectionctrl.NewClassSectionController(db)
	ucsH := sectionctrl.NewStudentClassSectionController(db)

	// Guard global: Admin/DKM + school admin check
	base := api.Group("",
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola class sections"),
			constants.AdminAndAbove,
		),
		schoolkuMiddleware.IsSchoolAdmin(),
	)

	// ========== 1) GENERIC (konteks via Header/Query/Host/Token) ==========
	base.Post("/class-sections", sectionH.CreateClassSection)
	base.Patch("/class-sections/:id", sectionH.UpdateClassSection)
	base.Delete("/class-sections/:id", sectionH.SoftDeleteClassSection)

	base.Post("/student-class-sections", ucsH.Create)
	base.Patch("/student-class-sections/:id", ucsH.Patch)
	base.Delete("/student-class-sections/:id", ucsH.Delete)

	// ========== 2) PATH-SCOPED by school_id ==========
	base.Post("/:school_id/class-sections", sectionH.CreateClassSection)
	base.Patch("/:school_id/class-sections/:id", sectionH.UpdateClassSection)
	base.Delete("/:school_id/class-sections/:id", sectionH.SoftDeleteClassSection)

	base.Post("/:school_id/student-class-sections", ucsH.Create)
	base.Get("/:school_id/student-class-sections/list", ucsH.ListAll)
	base.Patch("/:school_id/student-class-sections/:id", ucsH.Delete)
	base.Delete("/:school_id/student-class-sections/:id", ucsH.Delete)

	// ========== 3) PATH-SCOPED by school_slug ==========
	// base.Post("/m/:school_slug/class-sections", sectionH.CreateClassSection)
	// base.Patch("/m/:school_slug/class-sections/:id", sectionH.UpdateClassSection)
	// base.Delete("/m/:school_slug/class-sections/:id", sectionH.SoftDeleteClassSection)

	// base.Post("/m/:school_slug/student-class-sections", ucsH.Create)
	// base.Patch("/m/:school_slug/student-class-sections/:id", ucsH.Patch)
	// base.Delete("/m/:school_slug/student-class-sections/:id", ucsH.Delete)

	// ========== 4) JOIN-CODE (GENERIC) ==========
	base.Get("/class-sections/:id/join-code/student", sectionH.GetStudentJoinCode)
	base.Get("/class-sections/:id/join-code/teacher", sectionH.GetTeacherJoinCode)
	base.Get("/class-sections/:id/join-codes", sectionH.GetJoinCodes)

	base.Post("/class-sections/:id/join-code/student/rotate", sectionH.RotateStudentJoinCode)
	base.Post("/class-sections/:id/join-code/teacher/rotate", sectionH.RotateTeacherJoinCode)

	// ========== 5) JOIN-CODE (PATH-SCOPED by school_id) ==========
	base.Get("/:school_id/class-sections/:id/join-code/student", sectionH.GetStudentJoinCode)
	base.Get("/:school_id/class-sections/:id/join-code/teacher", sectionH.GetTeacherJoinCode)
	base.Get("/:school_id/class-sections/:id/join-codes", sectionH.GetJoinCodes)

	base.Post("/:school_id/class-sections/:id/join-code/student/rotate", sectionH.RotateStudentJoinCode)
	base.Post("/:school_id/class-sections/:id/join-code/teacher/rotate", sectionH.RotateTeacherJoinCode)

	// ========== 6) JOIN-CODE (PATH-SCOPED by school_slug) ==========
	// base.Get("/m/:school_slug/class-sections/:id/join-code/student", sectionH.GetStudentJoinCode)
	// base.Get("/m/:school_slug/class-sections/:id/join-code/teacher", sectionH.GetTeacherJoinCode)
	// base.Get("/m/:school_slug/class-sections/:id/join-codes", sectionH.GetJoinCodes)

	// base.Post("/m/:school_slug/class-sections/:id/join-code/student/rotate", sectionH.RotateStudentJoinCode)
	// base.Post("/m/:school_slug/class-sections/:id/join-code/teacher/rotate", sectionH.RotateTeacherJoinCode)

}
