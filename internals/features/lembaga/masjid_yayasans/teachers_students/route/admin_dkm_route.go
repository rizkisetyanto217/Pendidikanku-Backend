package route

import (
	"masjidku_backend/internals/constants"
	adminTeacherCtrl "masjidku_backend/internals/features/lembaga/masjid_yayasans/teachers_students/controller"
	authMiddleware "masjidku_backend/internals/middlewares/auth"
	masjidkuMiddleware "masjidku_backend/internals/middlewares/features"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func MasjidAdminRoutes(api fiber.Router, db *gorm.DB) {
	// ===== CONTROLLERS =====
	ctrlTeacher := adminTeacherCtrl.NewMasjidTeacherController(db) // teacher (baru refactor)
	v := validator.New()
	ctrlStudent := adminTeacherCtrl.New(db, v) // student controller

	// 🎓 /:masjid_id/masjid-teachers → DKM + Admin + Owner
	masjidTeachers := api.Group("/:masjid_id/masjid-teachers",
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola guru masjid"),
			constants.AdminAndAbove, // admin, dkm, owner
		),
		masjidkuMiddleware.IsMasjidAdmin(),
	)
	masjidTeachers.Post("/", ctrlTeacher.Create)
	masjidTeachers.Patch("/:id", ctrlTeacher.Update) // 🔥 tambahin update sesuai controller baru
	masjidTeachers.Delete("/:id", ctrlTeacher.Delete)

	// 🧑‍🎓 /:masjid_id/masjid-students → DKM + Admin + Owner
	masjidStudents := api.Group("/:masjid_id/masjid-students",
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola siswa/jamaah masjid"),
			constants.AdminAndAbove,
		),
		masjidkuMiddleware.IsMasjidAdmin(),
	)
	masjidStudents.Post("/", ctrlStudent.Create)
	masjidStudents.Put("/:id", ctrlStudent.Update)
	masjidStudents.Patch("/:id", ctrlStudent.Patch)
	masjidStudents.Delete("/:id", ctrlStudent.Delete)
	masjidStudents.Post("/:id/restore", ctrlStudent.Restore)
}