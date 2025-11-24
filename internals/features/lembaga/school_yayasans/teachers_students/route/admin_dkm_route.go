package route

import (
	"madinahsalam_backend/internals/constants"
	adminTeacherCtrl "madinahsalam_backend/internals/features/lembaga/school_yayasans/teachers_students/controller"
	authMiddleware "madinahsalam_backend/internals/middlewares/auth"
	schoolkuMiddleware "madinahsalam_backend/internals/middlewares/features"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func LembagaTeacherStudentAdminRoutes(api fiber.Router, db *gorm.DB) {
	// ===== CONTROLLERS =====
	ctrlTeacher := adminTeacherCtrl.NewSchoolTeacherController(db) // teacher (sudah pakai school dari token)
	v := validator.New()
	ctrlStudent := adminTeacherCtrl.New(db, v) // student controller

	// 🎓 /school-teachers → DKM + Admin + Owner
	// Controller akan:
	//   - ambil school_id dari token (ResolveSchoolIDFromContext)
	//   - pastikan DKM/Admin dengan EnsureDKMSchool di dalam controller
	schoolTeachers := api.Group("/school-teachers",
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola guru school"),
			constants.AdminAndAbove, // admin, dkm, owner
		),
		schoolkuMiddleware.IsSchoolAdmin(),
	)
	schoolTeachers.Post("/", ctrlTeacher.Create)
	schoolTeachers.Patch("/:id", ctrlTeacher.Update)
	schoolTeachers.Delete("/:id", ctrlTeacher.Delete)

	// 🧑‍🎓 /school-students → DKM + Admin + Owner
	// Student controller idealnya juga ambil school_id dari token
	// (pattern sama: ResolveSchoolIDFromContext + EnsureDKMSchool/EnsureStaffSchool di dalam).
	schoolStudents := api.Group("/school-students",
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola siswa/jamaah school"),
			constants.AdminAndAbove,
		),
		schoolkuMiddleware.IsSchoolAdmin(),
	)
	schoolStudents.Get("/list", ctrlStudent.List)
	schoolStudents.Post("/", ctrlStudent.Create)
	schoolStudents.Put("/:id", ctrlStudent.Update)
	schoolStudents.Patch("/:id", ctrlStudent.Patch)
	schoolStudents.Delete("/:id", ctrlStudent.Delete)
	schoolStudents.Post("/:id/restore", ctrlStudent.Restore)
}
