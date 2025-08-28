package model

import (
	UserModel "masjidku_backend/internals/features/users/user/model"
	"time"

	"gorm.io/gorm"
)

/* ===========================
   FAQ ANSWERS (child)
   =========================== */
type FaqAnswerModel struct {
	FaqAnswerID         string  `gorm:"column:faq_answer_id;primaryKey;type:uuid;default:gen_random_uuid()"`
	FaqAnswerQuestionID string  `gorm:"column:faq_answer_question_id;type:uuid;not null"`
	FaqAnswerAnsweredBy *string `gorm:"column:faq_answer_answered_by;type:uuid"` // NULLABLE (ON DELETE SET NULL)
	FaqAnswerText       string  `gorm:"column:faq_answer_text;type:text;not null"`

	// 🔗 Masjid
	FaqAnswerMasjidID string `gorm:"column:faq_answer_masjid_id;type:uuid;not null"`

	FaqAnswerCreatedAt time.Time      `gorm:"column:faq_answer_created_at;autoCreateTime"`
	FaqAnswerUpdatedAt time.Time      `gorm:"column:faq_answer_updated_at;autoUpdateTime"`
	FaqAnswerDeletedAt gorm.DeletedAt `gorm:"column:faq_answer_deleted_at;index"`

	// Relations
	User     *UserModel.UserModel  `gorm:"foreignKey:FaqAnswerAnsweredBy;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	Question *FaqQuestionModel     `gorm:"foreignKey:FaqAnswerQuestionID;references:FaqQuestionID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (FaqAnswerModel) TableName() string { return "faq_answers" }