// internals/features/lembaga/classes/sections/main/route/admin_route.go
package route

import (
	"madinahsalam_backend/internals/constants"
	sectionctrl "madinahsalam_backend/internals/features/school/classes/class_sections/controller"
	authMiddleware "madinahsalam_backend/internals/middlewares/auth"
	schoolkuMiddleware "madinahsalam_backend/internals/middlewares/features"

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

	// ==========================
	// 1) CLASS SECTIONS (admin)
	// ==========================
	classSec := base.Group("/class-sections")

	// CRUD section
	classSec.Post("/", sectionH.CreateClassSection)
	classSec.Patch("/:id", sectionH.UpdateClassSection)
	classSec.Delete("/:id", sectionH.SoftDeleteClassSection)

	// JOIN CODE subgroup: /class-sections/:id/join-code/...
	joinCode := classSec.Group("/:id/join-code")

	joinCode.Get("/student", sectionH.GetStudentJoinCode)
	joinCode.Get("/teacher", sectionH.GetTeacherJoinCode)
	joinCode.Get("/all", sectionH.GetJoinCodes) // path baru tapi tetap bisa kalau mau alias

	// atau kalau mau tetap path lama persis:
	// classSec.Get("/:id/join-code/student", sectionH.GetStudentJoinCode)
	// classSec.Get("/:id/join-code/teacher", sectionH.GetTeacherJoinCode)
	// classSec.Get("/:id/join-codes", sectionH.GetJoinCodes)

	joinCode.Post("/student/rotate", sectionH.RotateStudentJoinCode)
	joinCode.Post("/teacher/rotate", sectionH.RotateTeacherJoinCode)

	// ===================================
	// 2) STUDENT–CLASS SECTIONS (admin)
	// ===================================
	stuClassSec := base.Group("/student-class-sections")

	stuClassSec.Post("/", ucsH.Create)
	stuClassSec.Get("/list", ucsH.List)
	stuClassSec.Patch("/:id", ucsH.Patch)
	stuClassSec.Delete("/:id", ucsH.Delete)
}
