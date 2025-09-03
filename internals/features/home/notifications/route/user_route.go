package route

import (
	"masjidku_backend/internals/constants"
	"masjidku_backend/internals/features/home/notifications/controller"
	authMiddleware "masjidku_backend/internals/middlewares/auth"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Wajib login; semua role yang diizinkan
func NotificationUserRoutes(user fiber.Router, db *gorm.DB) {
	r := user.Group("/",
		authMiddleware.OnlyRolesSlice(
			"❌ Hanya pengguna terautentikasi yang boleh mengakses fitur notifikasi.",
			constants.AllowedRoles,
		),
	)

	// Controller Notifikasi untuk User (self scope)
	notifUserCtrl := controller.NewNotificationUserController(db)
	u := r.Group("/notification-users")
	u.Post("/read", notifUserCtrl.MarkAsRead) // tandai baca (hanya milik sendiri)

	// Jika kamu pisahkan controller view untuk user:
	nCtrl := controller.NewNotificationController(db)
	notifications := r.Group("/notifications")
	notifications.Get("/", nCtrl.GetAllNotificationsForUser) // semua notifikasi milik user
	notifications.Post("/by-masjid", nCtrl.GetNotificationsByMasjid) // filter berdasar masjid (opsional)
}
