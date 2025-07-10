package dto

import (
	"masjidku_backend/internals/features/home/notifications/model"

	"github.com/google/uuid"
)

// ================== REQUEST ==================
type NotificationRequest struct {
	NotificationTitle       string     `json:"notification_title"`
	NotificationDescription string     `json:"notification_description"`
	NotificationType        int        `json:"notification_type"`
	NotificationMasjidID    *uuid.UUID `json:"notification_masjid_id"` // nullable
	NotificationTags        []string   `json:"notification_tags"`      // optional
}

// ================== RESPONSE ==================
type NotificationResponse struct {
	NotificationID          uuid.UUID  `json:"notification_id"`
	NotificationTitle       string     `json:"notification_title"`
	NotificationDescription string     `json:"notification_description"`
	NotificationType        int        `json:"notification_type"`
	NotificationMasjidID    *uuid.UUID `json:"notification_masjid_id"` // nullable
	NotificationTags        []string   `json:"notification_tags"`
	NotificationCreatedAt   string     `json:"notification_created_at"`
	NotificationUpdatedAt   string     `json:"notification_updated_at"`
}

// ================ CONVERSION =================
func (r *NotificationRequest) ToModel() *model.NotificationModel {
	return &model.NotificationModel{
		NotificationTitle:       r.NotificationTitle,
		NotificationDescription: r.NotificationDescription,
		NotificationType:        r.NotificationType,
		NotificationMasjidID:    r.NotificationMasjidID,
		NotificationTags:        r.NotificationTags,
	}
}

func ToNotificationResponse(m *model.NotificationModel) *NotificationResponse {
	return &NotificationResponse{
		NotificationID:          m.NotificationID,
		NotificationTitle:       m.NotificationTitle,
		NotificationDescription: m.NotificationDescription,
		NotificationType:        m.NotificationType,
		NotificationMasjidID:    m.NotificationMasjidID,
		NotificationTags:        m.NotificationTags,
		NotificationCreatedAt:   m.NotificationCreatedAt.Format("2006-01-02 15:04:05"),
		NotificationUpdatedAt:   m.NotificationUpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

func ToNotificationResponseList(models []model.NotificationModel) []NotificationResponse {
	var result []NotificationResponse
	for _, m := range models {
		result = append(result, *ToNotificationResponse(&m))
	}
	return result
}
