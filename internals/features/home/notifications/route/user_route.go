package route

import (
	"masjidku_backend/internals/features/home/notifications/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func NotificationUserRoutes(user fiber.Router, db *gorm.DB) {
	ctrl := controller.NewNotificationController(db)

	notification := user.Group("/notifications")
	notification.Get("/", ctrl.GetAllNotificationsForUser) // 🟢 Lihat semua notifikasi untuk user
	notification.Post("/by-masjid", ctrl.GetNotificationsByMasjid)

}
