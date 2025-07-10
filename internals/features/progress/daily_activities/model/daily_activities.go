package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserDailyActivity struct {
	UserDailyActivityID           uint      `gorm:"column:user_daily_activity_id;primaryKey" json:"user_daily_activity_id"`
	UserDailyActivityUserID       uuid.UUID `gorm:"column:user_daily_activity_user_id;type:uuid;not null;index" json:"user_daily_activity_user_id"`
	UserDailyActivityActivityDate time.Time `gorm:"column:user_daily_activity_activity_date;type:date;not null;uniqueIndex:idx_user_activity_date" json:"user_daily_activity_activity_date"`
	UserDailyActivityAmountDay    int       `gorm:"column:user_daily_activity_amount_day;not null;default:1" json:"user_daily_activity_amount_day"`

	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index" json:"deleted_at,omitempty"`
}

// TableName override nama tabel
func (UserDailyActivity) TableName() string {
	return "user_daily_activities"
}
