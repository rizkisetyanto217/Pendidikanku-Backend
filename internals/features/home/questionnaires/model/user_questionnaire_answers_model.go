package model

import (
	UserModel "masjidku_backend/internals/features/users/user/model"
	"time"
)

type UserQuestionnaireAnswerModel struct {
	UserQuestionnaireID         string    `gorm:"column:user_questionnaire_id;primaryKey;type:uuid;default:gen_random_uuid()"`
	UserQuestionnaireUserID     string    `gorm:"column:user_questionnaire_user_id;type:uuid;not null"`
	UserQuestionnaireType       int       `gorm:"column:user_questionnaire_type;not null"` // 1=lecture, 2=event
	UserQuestionnaireRefID      *string   `gorm:"column:user_questionnaire_reference_id;type:uuid"`
	UserQuestionnaireQuestionID *string   `gorm:"column:user_questionnaire_question_id;type:uuid"`
	UserQuestionnaireAnswer     string    `gorm:"column:user_questionnaire_answer;type:text;not null"`
	UserQuestionnaireCreatedAt  time.Time `gorm:"column:user_questionnaire_created_at;autoCreateTime"`

	// Relations
	User     *UserModel.UserModel        `gorm:"foreignKey:UserQuestionnaireUserID"`
	Question *QuestionnaireQuestionModel `gorm:"foreignKey:UserQuestionnaireQuestionID"` // ⬅️ Tambahkan jika pakai preload
}

func (UserQuestionnaireAnswerModel) TableName() string {
	return "user_questionnaire_answers"
}
