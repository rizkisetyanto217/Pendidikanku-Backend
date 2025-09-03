package route

import (
	"masjidku_backend/internals/constants"
	adminTeacherCtrl "masjidku_backend/internals/features/lembaga/masjid_teachers/admins_teachers/controller"
	authMiddleware "masjidku_backend/internals/middlewares/auth"
	masjidkuMiddleware "masjidku_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func MasjidAdminRoutes(api fiber.Router, db *gorm.DB) {
	ctrlTeacher := adminTeacherCtrl.NewMasjidTeacherController(db) // inject db ke controller (admin)

	// 🎓 /masjid-teachers → DKM + Admin + Owner
	masjidTeachers := api.Group("/masjid-teachers",
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola guru masjid"),
			constants.AdminAndAbove, // admin, dkm, owner
		),
		masjidkuMiddleware.IsMasjidAdmin(), // scoping masjid_id dari token
	)

	masjidTeachers.Post("/",         ctrlTeacher.Create)
	masjidTeachers.Get("/by-masjid", ctrlTeacher.GetByMasjid)
	masjidTeachers.Delete("/:id",    ctrlTeacher.Delete)
}
