package route

import (
	"masjidku_backend/internals/constants"
	adminTeacherCtrl "masjidku_backend/internals/features/lembaga/teachers_students/controller"
	authMiddleware "masjidku_backend/internals/middlewares/auth"
	masjidkuMiddleware "masjidku_backend/internals/middlewares/features"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func MasjidAdminRoutes(api fiber.Router, db *gorm.DB) {
	// ===== CONTROLLERS =====
	ctrlTeacher := adminTeacherCtrl.NewMasjidTeacherController(db) // teacher (lama)
	v := validator.New()
	ctrlStudent := adminTeacherCtrl.New(db, v) // student (baru) – butuh validator

	// 🎓 /masjid-teachers → DKM + Admin + Owner
	masjidTeachers := api.Group("/masjid-teachers",
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola guru masjid"),
			constants.AdminAndAbove, // admin, dkm, owner
		),
		masjidkuMiddleware.IsMasjidAdmin(), // scoping masjid_id dari token
	)
	masjidTeachers.Post("/", ctrlTeacher.Create)
	masjidTeachers.Get("/list", ctrlTeacher.ListTeachers)
	masjidTeachers.Delete("/:id", ctrlTeacher.Delete)

	// 🧑‍🎓 /masjid-students → DKM + Admin + Owner
	masjidStudents := api.Group("/masjid-students",
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola siswa/jamaah masjid"),
			constants.AdminAndAbove,
		),
		masjidkuMiddleware.IsMasjidAdmin(),
	)
	masjidStudents.Post("/", ctrlStudent.Create)
	masjidStudents.Get("/", ctrlStudent.List)
	masjidStudents.Get("/list", ctrlStudent.List) // alias, mengikuti pola teachers
	masjidStudents.Get("/:id", ctrlStudent.GetByID)
	masjidStudents.Put("/:id", ctrlStudent.Update)
	masjidStudents.Patch("/:id", ctrlStudent.Patch)
	masjidStudents.Delete("/:id", ctrlStudent.Delete)
	masjidStudents.Post("/:id/restore", ctrlStudent.Restore)
}
