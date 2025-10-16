// file: internals/features/masjids/masjids/route/admin_dkm_route.go
package route

import (
	"masjidku_backend/internals/constants"
	"masjidku_backend/internals/middlewares/auth"

	masjidctl "masjidku_backend/internals/features/lembaga/masjid_yayasans/masjids/controller"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func MasjidAdminRoutes(admin fiber.Router, db *gorm.DB) {
	masjidCtrl := masjidctl.NewMasjidController(db, validator.New(), nil)
	profileCtrl := masjidctl.NewMasjidProfileController(db, validator.New())
	planCtrl := masjidctl.NewMasjidServicePlanController(db, validator.New())

	guard := auth.OnlyRolesSlice(
		constants.RoleErrorAdmin("aksi ini untuk admin/owner"),
		constants.AdminAndAbove,
	)

	// 🕌 MASJID (Admin/DKM) — pakai :masjid_id
	masjids := admin.Group("/masjids")
	masjids.Post("/", guard, masjidCtrl.CreateMasjidDKM)
	masjids.Get("/:masjid_id/get-teacher-code", guard, masjidCtrl.GetTeacherCode)
	masjids.Patch("/:masjid_id/teacher-code", guard, masjidCtrl.PatchTeacherCode)
	masjids.Put("/:masjid_id", guard, masjidCtrl.Patch)
	masjids.Delete("/:masjid_id", guard, masjidCtrl.Delete)

	// 🏷️ MASJID PROFILE (Admin/DKM)
	profilesByID := admin.Group("/:masjid_id/masjid-profiles", guard)
	profilesByID.Post("/", profileCtrl.Create)
	profilesByID.Patch("/:id", profileCtrl.Update)
	profilesByID.Delete("/:id", profileCtrl.Delete)

	// 🧩 SERVICE PLANS (Admin/Owner)
	alias := admin.Group("/masjid-service-plans", guard)
	alias.Get("/", planCtrl.List)
	alias.Post("/", planCtrl.Create)
	alias.Patch("/:id", planCtrl.Patch)
	alias.Delete("/:id", planCtrl.Delete)
}
