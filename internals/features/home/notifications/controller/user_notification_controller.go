package controller

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"masjidku_backend/internals/features/home/notifications/dto"
	"masjidku_backend/internals/features/home/notifications/model"
	userModel "masjidku_backend/internals/features/users/user/model"
)

type NotificationUserController struct {
	DB *gorm.DB
}

func NewNotificationUserController(db *gorm.DB) *NotificationUserController {
	return &NotificationUserController{DB: db}
}

// 🟢 POST /api/a/notification-users
func (ctrl *NotificationUserController) CreateNotificationUser(c *fiber.Ctx) error {
	var req dto.NotificationUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Permintaan tidak valid", "error": err.Error()})
	}

	notifUser := req.ToModel()
	if err := ctrl.DB.Create(notifUser).Error; err != nil {
		log.Printf("[ERROR] Gagal menyimpan notifikasi user: %v", err)
		return c.Status(500).JSON(fiber.Map{"message": "Gagal menyimpan notifikasi user", "error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message": "Notifikasi user berhasil dibuat",
		"data":    dto.ToNotificationUserResponse(notifUser),
	})
}

// 🟢 POST /api/a/notification-users/by-user
func (ctrl *NotificationUserController) GetNotificationsByUser(c *fiber.Ctx) error {
	type Payload struct {
		UserID uuid.UUID `json:"user_id"`
	}
	var payload Payload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Payload tidak valid", "error": err.Error()})
	}

	var notifUsers []model.NotificationUserModel
	if err := ctrl.DB.Where("notification_users_user_id = ?", payload.UserID).Find(&notifUsers).Error; err != nil {
		log.Printf("[ERROR] Gagal mengambil data notifikasi user: %v", err)
		return c.Status(500).JSON(fiber.Map{"message": "Gagal mengambil data notifikasi user"})
	}

	return c.JSON(fiber.Map{
		"message": "Berhasil mengambil notifikasi untuk user",
		"data":    dto.ToNotificationUserResponseList(notifUsers),
	})
}

// 🟢 PUT /api/a/notification-users/:id/read
func (ctrl *NotificationUserController) MarkAsRead(c *fiber.Ctx) error {
	id := c.Params("id")
	notifID, err := uuid.Parse(id)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "ID tidak valid", "error": err.Error()})
	}

	readTime := time.Now()
	if err := ctrl.DB.Model(&model.NotificationUserModel{}).
		Where("notification_users_id = ?", notifID).
		Updates(map[string]interface{}{
			"notification_users_read":    true,
			"notification_users_read_at": readTime,
		}).Error; err != nil {
		log.Printf("[ERROR] Gagal mengupdate notifikasi sebagai dibaca: %v", err)
		return c.Status(500).JSON(fiber.Map{"message": "Gagal update notifikasi"})
	}

	return c.JSON(fiber.Map{"message": "Notifikasi ditandai sebagai dibaca"})
}

// 🟢 POST /api/a/notification-users/broadcast
// 🟢 POST /api/a/notification-users/broadcast
func (ctrl *NotificationUserController) BroadcastToAllUsers(c *fiber.Ctx) error {
	type Payload struct {
		NotificationID uuid.UUID `json:"notification_id"`
	}

	var payload Payload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Payload tidak valid", "error": err.Error()})
	}

	// Ambil semua user ID dari tabel users
	var userIDs []uuid.UUID
	if err := ctrl.DB.Model(&userModel.UserModel{}).Pluck("id", &userIDs).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Gagal mengambil data user", "error": err.Error()})
	}

	// Buat data notification_user untuk tiap user
	var notifUsers []model.NotificationUserModel
	now := time.Now()
	for _, uid := range userIDs {
		notifUsers = append(notifUsers, model.NotificationUserModel{
			NotificationUserNotificationID: payload.NotificationID,
			NotificationUserUserID:         uid,
			NotificationUserSentAt:         now,
			NotificationUserRead:           false,
		})
	}

	// Bulk insert (gunakan batch untuk efisiensi)
	if err := ctrl.DB.CreateInBatches(&notifUsers, 1000).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Gagal mengirim notifikasi massal", "error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message":     "Berhasil mengirim notifikasi ke semua user",
		"jumlah_user": len(userIDs),
	})
}
