package dto

import (
	"masjidku_backend/internals/features/home/notifications/model"
	"time"

	"github.com/google/uuid"
)

// 🟢 Request DTO untuk membuat data notification_user
type NotificationUserRequest struct {
	NotificationID uuid.UUID  `json:"notification_users_notification_id"`
	UserID         uuid.UUID  `json:"notification_users_user_id"`
	Read           bool       `json:"notification_users_read"`
	ReadAt         *time.Time `json:"notification_users_read_at,omitempty"` // opsional
}

// 🔵 Response DTO untuk mengirim ke frontend
type NotificationUserResponse struct {
	ID             uuid.UUID `json:"notification_users_id"`
	NotificationID uuid.UUID `json:"notification_users_notification_id"`
	UserID         uuid.UUID `json:"notification_users_user_id"`
	Read           bool      `json:"notification_users_read"`
	SentAt         string    `json:"notification_users_sent_at"`
	ReadAt         *string   `json:"notification_users_read_at,omitempty"`
}

// 🔄 Konversi dari Request ke Model
func (r *NotificationUserRequest) ToModel() *model.NotificationUserModel {
	return &model.NotificationUserModel{
		NotificationUserNotificationID: r.NotificationID,
		NotificationUserUserID:         r.UserID,
		NotificationUserRead:           r.Read,
		NotificationUserReadAt:         r.ReadAt,
	}
}

// 🔄 Konversi dari Model ke Response
func ToNotificationUserResponse(m *model.NotificationUserModel) *NotificationUserResponse {
	var readAt *string
	if m.NotificationUserReadAt != nil {
		formatted := m.NotificationUserReadAt.Format("2006-01-02 15:04:05")
		readAt = &formatted
	}

	return &NotificationUserResponse{
		ID:             m.NotificationUserID,
		NotificationID: m.NotificationUserNotificationID,
		UserID:         m.NotificationUserUserID,
		Read:           m.NotificationUserRead,
		SentAt:         m.NotificationUserSentAt.Format("2006-01-02 15:04:05"),
		ReadAt:         readAt,
	}
}

// 🔁 Convert slice of models to slice of response DTOs
func ToNotificationUserResponseList(models []model.NotificationUserModel) []NotificationUserResponse {
	var responses []NotificationUserResponse
	for _, m := range models {
		responses = append(responses, *ToNotificationUserResponse(&m))
	}
	return responses
}
