// file: internals/features/masjids/masjids/route/admin_dkm_route.go
package route

import (
	"masjidku_backend/internals/features/lembaga/masjids/controller"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func MasjidUserRoutes(admin fiber.Router, db *gorm.DB) {
	masjidCtrl  := controller.NewMasjidController(db, validator.New(), nil)

	// =========================
	// 🕌 MASJID
	// =========================

	// Prefix: /masjids
	masjids := admin.Group("/masjids")

	// OWNER-only untuk aksi sensitif/lintas tenant → /api/a/masjids/owner/...
	masjidsOwner := masjids.Group("/user")
	masjidsOwner.Post("/", masjidCtrl.CreateMasjidDKM)

}