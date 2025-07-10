package route

import (
	"masjidku_backend/internals/features/home/notifications/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AllNotificationRoutes(api fiber.Router, db *gorm.DB) {
	notifCtrl := controller.NewNotificationController(db)

	notif := api.Group("/notifications")
	notif.Post("/", notifCtrl.CreateNotification)
	notif.Get("/", notifCtrl.GetAllNotifications)
	

	// 🔹 Controller Notifikasi untuk User
	notifUserCtrl := controller.NewNotificationUserController(db)
	notifUser := api.Group("/notification-users")
	notifUser.Post("/", notifUserCtrl.CreateNotificationUser)
	notifUser.Post("/by-user", notifUserCtrl.GetNotificationsByUser)
	notifUser.Post("/read", notifUserCtrl.MarkAsRead)
	notifUser.Post("/broadcast", notifUserCtrl.BroadcastToAllUsers)
}
