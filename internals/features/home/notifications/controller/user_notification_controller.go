package controller

import (
	"log"
	"strconv"
	"time"

	"masjidku_backend/internals/features/home/notifications/dto"
	"masjidku_backend/internals/features/home/notifications/model"
	userModel "masjidku_backend/internals/features/users/user/model"
	helper "masjidku_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
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
		return helper.JsonError(c, fiber.StatusBadRequest, "Permintaan tidak valid")
	}

	notifUser := req.ToModel()
	if err := ctrl.DB.Create(notifUser).Error; err != nil {
		log.Printf("[ERROR] create notification_user: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan notifikasi user")
	}

	return helper.JsonCreated(c, "Notifikasi user berhasil dibuat", dto.ToNotificationUserResponse(notifUser))
}

// 🟢 POST /api/a/notification-users/by-user   (body: { "user_id": "..." }) + pagination
func (ctrl *NotificationUserController) GetNotificationsByUser(c *fiber.Ctx) error {
	type Payload struct {
		UserID uuid.UUID `json:"user_id"`
	}
	var payload Payload
	if err := c.BodyParser(&payload); err != nil || payload.UserID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}

	// pagination
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	// count
	var total int64
	if err := ctrl.DB.Model(&model.NotificationUserModel{}).
		Where("notification_users_user_id = ?", payload.UserID).
		Count(&total).Error; err != nil {
		log.Printf("[ERROR] count notification_users: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data notifikasi user")
	}

	// data
	var notifUsers []model.NotificationUserModel
	if err := ctrl.DB.
		Where("notification_users_user_id = ?", payload.UserID).
		Order("notification_users_sent_at DESC").
		Limit(limit).Offset(offset).
		Find(&notifUsers).Error; err != nil {
		log.Printf("[ERROR] get notification_users: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data notifikasi user")
	}

	pagination := fiber.Map{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": int((total + int64(limit) - 1) / int64(limit)),
		"has_next":    int64(page*limit) < total,
		"has_prev":    page > 1,
		"user_id":     payload.UserID,
	}

	return helper.JsonList(c, dto.ToNotificationUserResponseList(notifUsers), pagination)
}

// 🟢 PUT /api/a/notification-users/:id/read
func (ctrl *NotificationUserController) MarkAsRead(c *fiber.Ctx) error {
	id := c.Params("id")
	notifID, err := uuid.Parse(id)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	readTime := time.Now()
	if err := ctrl.DB.Model(&model.NotificationUserModel{}).
		Where("notification_users_id = ?", notifID).
		Updates(map[string]interface{}{
			"notification_users_read":    true,
			"notification_users_read_at": readTime,
		}).Error; err != nil {
		log.Printf("[ERROR] mark read notification_user: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal update notifikasi")
	}

	return helper.JsonOK(c, "Notifikasi ditandai sebagai dibaca", fiber.Map{"id": notifID})
}

// 🟢 POST /api/a/notification-users/broadcast
func (ctrl *NotificationUserController) BroadcastToAllUsers(c *fiber.Ctx) error {
	type Payload struct {
		NotificationID uuid.UUID `json:"notification_id"`
	}
	var payload Payload
	if err := c.BodyParser(&payload); err != nil || payload.NotificationID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}

	// ambil semua user id
	var userIDs []uuid.UUID
	if err := ctrl.DB.Model(&userModel.UserModel{}).Pluck("id", &userIDs).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data user")
	}

	// bulk insert notification_users
	now := time.Now()
	notifUsers := make([]model.NotificationUserModel, 0, len(userIDs))
	for _, uid := range userIDs {
		notifUsers = append(notifUsers, model.NotificationUserModel{
			NotificationUserNotificationID: payload.NotificationID,
			NotificationUserUserID:         uid,
			NotificationUserSentAt:         now,
			NotificationUserRead:           false,
		})
	}
	if err := ctrl.DB.CreateInBatches(&notifUsers, 1000).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengirim notifikasi massal")
	}

	return helper.JsonOK(c, "Berhasil mengirim notifikasi ke semua user", fiber.Map{
		"jumlah_user": len(userIDs),
	})
}
