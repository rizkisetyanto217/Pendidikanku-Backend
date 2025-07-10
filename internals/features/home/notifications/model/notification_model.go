package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type NotificationModel struct {
	NotificationID          uuid.UUID      `gorm:"column:notification_id;primaryKey;type:uuid;default:gen_random_uuid()" json:"notification_id"`
	NotificationTitle       string         `gorm:"column:notification_title;type:varchar(255);not null" json:"notification_title"`
	NotificationDescription string         `gorm:"column:notification_description;type:text" json:"notification_description"`
	NotificationType        int            `gorm:"column:notification_type;not null" json:"notification_type"`            // Di-handle enum di sisi kode
	NotificationMasjidID    *uuid.UUID     `gorm:"column:notification_masjid_id;type:uuid" json:"notification_masjid_id"` // nullable
	NotificationTags        pq.StringArray `gorm:"column:notification_tags;type:text[]" json:"notification_tags"`
	NotificationCreatedAt   time.Time      `gorm:"column:notification_created_at;autoCreateTime" json:"notification_created_at"`
	NotificationUpdatedAt   time.Time      `gorm:"column:notification_updated_at;autoUpdateTime" json:"notification_updated_at"`
}

func (NotificationModel) TableName() string {
	return "notifications"
}