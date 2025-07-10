package controller

import (
	"log"
	"masjidku_backend/internals/features/home/notifications/dto"
	"masjidku_backend/internals/features/home/notifications/model"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NotificationController struct {
	DB *gorm.DB
}

func NewNotificationController(db *gorm.DB) *NotificationController {
	return &NotificationController{DB: db}
}

// 🟢 POST /api/a/notifications
func (ctrl *NotificationController) CreateNotification(c *fiber.Ctx) error {
	var req dto.NotificationRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("[ERROR] Gagal parsing body: %v", err)
		return c.Status(400).JSON(fiber.Map{
			"message": "Permintaan tidak valid",
			"error":   err.Error(),
		})
	}

	newNotif := req.ToModel()
	if err := ctrl.DB.Create(newNotif).Error; err != nil {
		log.Printf("[ERROR] Gagal menyimpan notifikasi: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"message": "Gagal menyimpan notifikasi",
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Notifikasi berhasil dikirim",
		"data":    dto.ToNotificationResponse(newNotif),
	})
}

// 🟢 POST /api/a/notifications/by-masjid
func (ctrl *NotificationController) GetNotificationsByMasjid(c *fiber.Ctx) error {
	type MasjidPayload struct {
		MasjidID uuid.UUID `json:"masjid_id"`
	}

	var payload MasjidPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Payload tidak valid", "error": err.Error()})
	}

	var notifs []model.NotificationModel
	if err := ctrl.DB.Where("notification_masjid_id = ?", payload.MasjidID).Find(&notifs).Error; err != nil {
		log.Printf("[ERROR] Gagal mengambil data notifikasi: %v", err)
		return c.Status(500).JSON(fiber.Map{"message": "Gagal mengambil data"})
	}

	return c.JSON(fiber.Map{
		"message": "Berhasil mengambil notifikasi masjid",
		"data":    dto.ToNotificationResponseList(notifs),
	})
}

// 🟢 GET /api/a/notifications
func (ctrl *NotificationController) GetAllNotifications(c *fiber.Ctx) error {
	var notifs []model.NotificationModel
	if err := ctrl.DB.Order("notification_created_at desc").Find(&notifs).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Gagal mengambil data"})
	}

	return c.JSON(fiber.Map{
		"message": "Berhasil mengambil semua notifikasi",
		"data":    dto.ToNotificationResponseList(notifs),
	})
}

// 🟢 GET /api/u/notifications
func (ctrl *NotificationController) GetAllNotificationsForUser(c *fiber.Ctx) error {
	var notifs []model.NotificationModel
	if err := ctrl.DB.Order("notification_created_at desc").Find(&notifs).Error; err != nil {
		log.Printf("[ERROR] Gagal mengambil notifikasi untuk user: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"message": "Gagal mengambil notifikasi",
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Berhasil mengambil notifikasi untuk user",
		"data":    dto.ToNotificationResponseList(notifs),
	})
}
