package model

import (
	"time"

	"github.com/lib/pq"
)

type SurveyQuestion struct {
	SurveyQuestionID         int            `gorm:"column:survey_question_id;primaryKey" json:"survey_question_id"`
	SurveyQuestionText       string         `gorm:"column:survey_question_text;type:text;not null" json:"survey_question_text"`
	SurveyQuestionAnswer     pq.StringArray `gorm:"column:survey_question_answer;type:text[]" json:"survey_question_answer,omitempty"`
	SurveyQuestionOrderIndex int            `gorm:"column:survey_question_order_index;not null;index" json:"survey_question_order_index"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (SurveyQuestion) TableName() string {
	return "survey_questions"
}