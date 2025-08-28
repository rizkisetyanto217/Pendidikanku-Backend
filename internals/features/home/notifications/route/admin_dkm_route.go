package route

import (
	"masjidku_backend/internals/constants"
	"masjidku_backend/internals/features/home/notifications/controller"
	authMiddleware "masjidku_backend/internals/middlewares/auth"
	masjidkuMiddleware "masjidku_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Login wajib + role admin/dkm/owner + scope masjid
func NotificationAdminRoutes(api fiber.Router, db *gorm.DB) {
	admin := api.Group("/",
		authMiddleware.AuthMiddleware(db),
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola notifikasi"),
			constants.AdminAndAbove, // admin, dkm, owner
		),
		masjidkuMiddleware.IsMasjidAdmin(), // inject/cek masjid_id dari token
	)

	// --- Notifikasi (master) ---
	notifCtrl := controller.NewNotificationController(db)
	notif := admin.Group("/notifications")
	notif.Post("/", notifCtrl.CreateNotification)  // buat notifikasi (scoped masjid)
	notif.Get("/", notifCtrl.GetAllNotifications)  // list internal (dashboard admin)
	notif.Delete("/:id", notifCtrl.DeleteNotification)

	// --- Notifikasi-User (penugasan/aksi admin) ---
	notifUserCtrl := controller.NewNotificationUserController(db)
	nUser := admin.Group("/notification-users")
	nUser.Post("/", notifUserCtrl.CreateNotificationUser)      // assign notifikasi ke user tertentu
	nUser.Post("/by-user", notifUserCtrl.GetNotificationsByUser) // laporan internal by user (opsional)
	nUser.Post("/broadcast", notifUserCtrl.BroadcastToAllUsers)  // broadcast ke semua user (scoped masjid)
}
