package route

import (
	"masjidku_backend/internals/constants"
	adminTeacherCtrl "masjidku_backend/internals/features/lembaga/masjid_admins_teachers/controller"
	authMiddleware "masjidku_backend/internals/middlewares/auth"
	masjidkuMiddleware "masjidku_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func MasjidAdminRoutes(api fiber.Router, db *gorm.DB) {
	ctrlAdmin := adminTeacherCtrl.NewMasjidAdminController(db)
	ctrlTeacher := adminTeacherCtrl.NewMasjidTeacherController(db)

	// 🛡️ /masjid-admins → OWNER only (tetap)
	masjidAdmins := api.Group("/masjid-admins/owner",
		authMiddleware.AuthMiddleware(db),
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorOwner("mengelola admin masjid"),
			constants.OwnerOnly,
		),
		masjidkuMiddleware.IsMasjidAdmin(), 
	)
	masjidAdmins.Post("/",          ctrlAdmin.AddAdmin)
	masjidAdmins.Post("/by-masjid", ctrlAdmin.GetAdminsByMasjid)
	masjidAdmins.Put("/revoke",     ctrlAdmin.RevokeAdmin)

	// 🎓 /masjid-teachers → DKM + Admin + Owner
	masjidTeachers := api.Group("/masjid-teachers",
		authMiddleware.AuthMiddleware(db),
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
