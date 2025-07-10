package model

import (
	"time"

	"github.com/google/uuid"
)

type UserSurvey struct {
	UserSurveyID         int       `gorm:"column:user_survey_id;primaryKey" json:"user_survey_id"`
	UserSurveyUserID     uuid.UUID `gorm:"column:user_survey_user_id;type:uuid;not null;index" json:"user_survey_user_id"`
	UserSurveyQuestionID int       `gorm:"column:user_survey_question_id;not null;index" json:"user_survey_question_id"`
	UserSurveyAnswer     string    `gorm:"column:user_survey_answer;type:text;not null" json:"user_survey_answer"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (UserSurvey) TableName() string {
	return "user_surveys"
}