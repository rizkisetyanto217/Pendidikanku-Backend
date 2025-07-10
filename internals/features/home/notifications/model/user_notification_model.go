package model

import (
	"time"

	"github.com/google/uuid"
)

type NotificationUserModel struct {
	NotificationUserID             uuid.UUID  `gorm:"column:notification_users_id;primaryKey;type:uuid;default:gen_random_uuid()" json:"notification_users_id"`
	NotificationUserNotificationID uuid.UUID  `gorm:"column:notification_users_notification_id;type:uuid;not null" json:"notification_users_notification_id"`
	NotificationUserUserID         uuid.UUID  `gorm:"column:notification_users_user_id;type:uuid;not null" json:"notification_users_user_id"`
	NotificationUserRead           bool       `gorm:"column:notification_users_read;default:false" json:"notification_users_read"`
	NotificationUserSentAt         time.Time  `gorm:"column:notification_users_sent_at;default:CURRENT_TIMESTAMP" json:"notification_users_sent_at"`
	NotificationUserReadAt         *time.Time `gorm:"column:notification_users_read_at" json:"notification_users_read_at"`
}

// TableName overrides the table name used by GORM
func (NotificationUserModel) TableName() string {
	return "notification_users"
}
