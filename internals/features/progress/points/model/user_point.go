package model

import (
	"time"

	"github.com/google/uuid"
)

type UserPointLog struct {
	UserPointLogID         uint      `gorm:"column:user_point_log_id;primaryKey" json:"user_point_log_id"`                   // ID unik log poin
	UserPointLogUserID     uuid.UUID `gorm:"column:user_point_log_user_id;type:uuid;not null" json:"user_point_log_user_id"` // UUID user
	UserPointLogPoints     int       `gorm:"column:user_point_log_points;not null" json:"user_point_log_points"`             // Jumlah poin
	UserPointLogSourceType int       `gorm:"column:user_point_log_source_type;not null" json:"user_point_log_source_type"`   // Tipe sumber
	UserPointLogSourceID   int       `gorm:"column:user_point_log_source_id" json:"user_point_log_source_id"`                // ID sumber
	CreatedAt              time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`                             // Timestamp
}

func (UserPointLog) TableName() string {
	return "user_point_logs"
}
