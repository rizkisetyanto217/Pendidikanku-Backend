package route

import (
	"masjidku_backend/internals/constants"
	ycontroller "masjidku_backend/internals/features/lembaga/yayasans/controller"
	"masjidku_backend/internals/middlewares/auth"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func YayasanAdminRoutes(admin fiber.Router, db *gorm.DB) {
	yayasanCtrl := ycontroller.NewYayasanController(db, nil, nil)

	// =========================
	// 🏢 YAYASAN (ADMIN AREA)
	// =========================

	// Prefix: /yayasans
	yayasans := admin.Group("/yayasans")

	// Admin/owner untuk operasi harian → /api/a/yayasans/...
	yayasansAdmin := yayasans.Group("/",
		auth.OnlyRolesSlice(
			constants.RoleErrorAdmin("aksi ini untuk admin/owner"),
			constants.AdminAndAbove,
		),
	)

	// CRUD + restore (soft-delete aware)
	yayasansAdmin.Post("/",      yayasanCtrl.Create)
	yayasansAdmin.Get("/",       yayasanCtrl.List)
	yayasansAdmin.Patch("/:id",  yayasanCtrl.Update)
	yayasansAdmin.Delete("/",    yayasanCtrl.Delete)  // by body
	yayasansAdmin.Delete("/:id", yayasanCtrl.Delete)  // by param
}
