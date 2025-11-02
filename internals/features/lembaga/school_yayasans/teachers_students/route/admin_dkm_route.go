package route

import (
	"schoolku_backend/internals/constants"
	adminTeacherCtrl "schoolku_backend/internals/features/lembaga/school_yayasans/teachers_students/controller"
	authMiddleware "schoolku_backend/internals/middlewares/auth"
	schoolkuMiddleware "schoolku_backend/internals/middlewares/features"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func SchoolAdminRoutes(api fiber.Router, db *gorm.DB) {
	// ===== CONTROLLERS =====
	ctrlTeacher := adminTeacherCtrl.NewSchoolTeacherController(db) // teacher (baru refactor)
	v := validator.New()
	ctrlStudent := adminTeacherCtrl.New(db, v) // student controller

	// 🎓 /:school_id/school-teachers → DKM + Admin + Owner
	schoolTeachers := api.Group("/:school_id/school-teachers",
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola guru school"),
			constants.AdminAndAbove, // admin, dkm, owner
		),
		schoolkuMiddleware.IsSchoolAdmin(),
	)
	schoolTeachers.Post("/", ctrlTeacher.Create)
	schoolTeachers.Patch("/:id", ctrlTeacher.Update) // 🔥 tambahin update sesuai controller baru
	schoolTeachers.Delete("/:id", ctrlTeacher.Delete)

	// 🧑‍🎓 /:school_id/school-students → DKM + Admin + Owner
	schoolStudents := api.Group("/:school_id/school-students",
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola siswa/jamaah school"),
			constants.AdminAndAbove,
		),
		schoolkuMiddleware.IsSchoolAdmin(),
	)
	schoolStudents.Post("/", ctrlStudent.Create)
	schoolStudents.Put("/:id", ctrlStudent.Update)
	schoolStudents.Patch("/:id", ctrlStudent.Patch)
	schoolStudents.Delete("/:id", ctrlStudent.Delete)
	schoolStudents.Post("/:id/restore", ctrlStudent.Restore)
}
