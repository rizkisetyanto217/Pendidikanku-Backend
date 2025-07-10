package model

import (
	"time"

	"github.com/google/uuid"
)

type UserProgress struct {
	UserProgressID          uint      `gorm:"column:user_progress_id;primaryKey" json:"user_progress_id"`
	UserProgressUserID      uuid.UUID `gorm:"column:user_progress_user_id;type:uuid;not null;unique" json:"user_progress_user_id"`
	UserProgressTotalPoints int       `gorm:"column:user_progress_total_points;not null;default:0" json:"user_progress_total_points"`
	UserProgressLevel       int       `gorm:"column:user_progress_level;not null;default:1" json:"user_progress_level"`
	UserProgressRank        int       `gorm:"column:user_progress_rank;not null;default:1" json:"user_progress_rank"`
	LastUpdated             time.Time `gorm:"column:last_updated;autoUpdateTime" json:"last_updated"`
}

func (UserProgress) TableName() string {
	return "user_progress"
}
