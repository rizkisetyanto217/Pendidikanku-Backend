package model

import (
	UserModel "masjidku_backend/internals/features/users/user/model"
	"time"
)

type FaqAnswerModel struct {
	FaqAnswerID         string    `gorm:"column:faq_answer_id;primaryKey;type:uuid;default:gen_random_uuid()"`
	FaqAnswerQuestionID string    `gorm:"column:faq_answer_question_id;type:uuid;not null"`
	FaqAnswerAnsweredBy string    `gorm:"column:faq_answer_answered_by;type:uuid;not null"`
	FaqAnswerText       string    `gorm:"column:faq_answer_text;type:text;not null"`
	FaqAnswerCreatedAt  time.Time `gorm:"column:faq_answer_created_at;autoCreateTime"`

	// Relations
	User     *UserModel.UserModel `gorm:"foreignKey:FaqAnswerAnsweredBy"`
	Question *FaqQuestionModel    `gorm:"foreignKey:FaqAnswerQuestionID"`
}

func (FaqAnswerModel) TableName() string {
	return "faq_answers"
}