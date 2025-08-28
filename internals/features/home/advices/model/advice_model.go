// =====================
// model/advice.go
// =====================

package model

import (
	"time"

	"gorm.io/gorm"
)

type AdviceModel struct {
	AdviceID          string         `gorm:"column:advice_id;primaryKey;type:uuid;default:gen_random_uuid()" json:"advice_id"`
	AdviceDescription string         `gorm:"column:advice_description;type:text;not null" json:"advice_description"`
	AdviceLectureID   *string        `gorm:"column:advice_lecture_id;type:uuid" json:"advice_lecture_id,omitempty"` // nullable
	AdviceUserID      string         `gorm:"column:advice_user_id;type:uuid;not null" json:"advice_user_id"`

	AdviceCreatedAt time.Time      `gorm:"column:advice_created_at;autoCreateTime" json:"advice_created_at"`
	AdviceUpdatedAt time.Time      `gorm:"column:advice_updated_at;autoUpdateTime" json:"advice_updated_at"`
	AdviceDeletedAt gorm.DeletedAt `gorm:"column:advice_deleted_at;index" json:"-"`
}

func (AdviceModel) TableName() string { return "advices" }
