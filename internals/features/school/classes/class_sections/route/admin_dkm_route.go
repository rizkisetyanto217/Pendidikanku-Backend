// internals/features/lembaga/classes/sections/main/route/admin_route.go
package route

import (
	"masjidku_backend/internals/constants"
	sectionctrl "masjidku_backend/internals/features/school/classes/class_sections/controller"
	authMiddleware "masjidku_backend/internals/middlewares/auth"
	masjidkuMiddleware "masjidku_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func ClassSectionAdminRoutes(api fiber.Router, db *gorm.DB) {
	// Controllers
	sectionH := sectionctrl.NewClassSectionController(db)
	ucsH := sectionctrl.NewStudentClassSectionController(db)

	// Guard global: Admin/DKM + masjid admin check
	base := api.Group("",
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola class sections"),
			constants.AdminAndAbove,
		),
		masjidkuMiddleware.IsMasjidAdmin(),
	)

	// ========== 1) GENERIC (konteks via Header/Query/Host/Token) ==========
	base.Post("/class-sections", sectionH.CreateClassSection)
	base.Patch("/class-sections/:id", sectionH.UpdateClassSection)
	base.Delete("/class-sections/:id", sectionH.SoftDeleteClassSection)

	base.Post("/student-class-sections", ucsH.Create)
	base.Patch("/student-class-sections/:id", ucsH.Patch)
	base.Delete("/student-class-sections/:id", ucsH.Delete)

	// ========== 2) PATH-SCOPED by masjid_id ==========
	base.Post("/:masjid_id/class-sections", sectionH.CreateClassSection)
	base.Patch("/:masjid_id/class-sections/:id", sectionH.UpdateClassSection)
	base.Delete("/:masjid_id/class-sections/:id", sectionH.SoftDeleteClassSection)

	base.Post("/:masjid_id/student-class-sections", ucsH.Create)
	base.Get("/:masjid_id/student-class-sections/list", ucsH.ListAll)
	base.Patch("/:masjid_id/student-class-sections/:id", ucsH.Delete)
	base.Delete("/:masjid_id/student-class-sections/:id", ucsH.Delete)

	// ========== 3) PATH-SCOPED by masjid_slug ==========
	// base.Post("/m/:masjid_slug/class-sections", sectionH.CreateClassSection)
	// base.Patch("/m/:masjid_slug/class-sections/:id", sectionH.UpdateClassSection)
	// base.Delete("/m/:masjid_slug/class-sections/:id", sectionH.SoftDeleteClassSection)

	// base.Post("/m/:masjid_slug/student-class-sections", ucsH.Create)
	// base.Patch("/m/:masjid_slug/student-class-sections/:id", ucsH.Patch)
	// base.Delete("/m/:masjid_slug/student-class-sections/:id", ucsH.Delete)

	// ========== 4) JOIN-CODE (GENERIC) ==========
	base.Get("/class-sections/:id/join-code/student", sectionH.GetStudentJoinCode)
	base.Get("/class-sections/:id/join-code/teacher", sectionH.GetTeacherJoinCode)
	base.Get("/class-sections/:id/join-codes", sectionH.GetJoinCodes)

	base.Post("/class-sections/:id/join-code/student/rotate", sectionH.RotateStudentJoinCode)
	base.Post("/class-sections/:id/join-code/teacher/rotate", sectionH.RotateTeacherJoinCode)

	// ========== 5) JOIN-CODE (PATH-SCOPED by masjid_id) ==========
	base.Get("/:masjid_id/class-sections/:id/join-code/student", sectionH.GetStudentJoinCode)
	base.Get("/:masjid_id/class-sections/:id/join-code/teacher", sectionH.GetTeacherJoinCode)
	base.Get("/:masjid_id/class-sections/:id/join-codes", sectionH.GetJoinCodes)

	base.Post("/:masjid_id/class-sections/:id/join-code/student/rotate", sectionH.RotateStudentJoinCode)
	base.Post("/:masjid_id/class-sections/:id/join-code/teacher/rotate", sectionH.RotateTeacherJoinCode)

	// ========== 6) JOIN-CODE (PATH-SCOPED by masjid_slug) ==========
	// base.Get("/m/:masjid_slug/class-sections/:id/join-code/student", sectionH.GetStudentJoinCode)
	// base.Get("/m/:masjid_slug/class-sections/:id/join-code/teacher", sectionH.GetTeacherJoinCode)
	// base.Get("/m/:masjid_slug/class-sections/:id/join-codes", sectionH.GetJoinCodes)

	// base.Post("/m/:masjid_slug/class-sections/:id/join-code/student/rotate", sectionH.RotateStudentJoinCode)
	// base.Post("/m/:masjid_slug/class-sections/:id/join-code/teacher/rotate", sectionH.RotateTeacherJoinCode)

}
