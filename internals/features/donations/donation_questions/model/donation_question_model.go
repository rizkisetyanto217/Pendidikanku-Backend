package model

import (
	"time"

	"gorm.io/gorm"
)

type DonationQuestionModel struct {
	DonationQuestionID             uint   `gorm:"column:donation_question_id;primaryKey" json:"donation_question_id"`                    // ID unik
	DonationQuestionDonationID     uint   `gorm:"column:donation_question_donation_id;not null" json:"donation_question_donation_id"`    // FK ke donations
	DonationQuestionQuestionID     uint   `gorm:"column:donation_question_question_id;not null" json:"donation_question_question_id"`    // FK ke questions
	DonationQuestionUserProgressID *uint  `gorm:"column:donation_question_user_progress_id" json:"donation_question_user_progress_id"`   // FK ke user_progress (nullable)
	DonationQuestionUserMessage    string `gorm:"column:donation_question_user_message;type:text" json:"donation_question_user_message"` // Optional pesan user

	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`  // Timestamp dibuat
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`  // Timestamp update
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index" json:"deleted_at,omitempty"` // Soft delete opsional
}

// TableName memastikan nama tabel sesuai
func (DonationQuestionModel) TableName() string {
	return "donation_questions"
}