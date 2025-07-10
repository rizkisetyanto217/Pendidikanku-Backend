package model

import "time"

type AdviceModel struct {
	AdviceID          string    `gorm:"column:advice_id;primaryKey;type:uuid;default:gen_random_uuid()" json:"advice_id"`
	AdviceDescription string    `gorm:"column:advice_description;type:text;not null"`
	AdviceLectureID   *string   `gorm:"column:advice_lecture_id;type:uuid"` // Nullable
	AdviceUserID      string    `gorm:"column:advice_user_id;type:uuid;not null"`
	AdviceCreatedAt   time.Time `gorm:"column:advice_created_at;autoCreateTime"`
}

// TableName sets the name of the table
func (AdviceModel) TableName() string {
	return "advices"
}
