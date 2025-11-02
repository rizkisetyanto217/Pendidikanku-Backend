package model

import (
	EventModel "schoolku_backend/internals/features/schools/events/model"
	LectureSessionModel "schoolku_backend/internals/features/schools/lecture_sessions/main/model"
	"time"
)

type QuestionnaireQuestionModel struct {
	QuestionID       string    `gorm:"column:questionnaire_question_id;primaryKey;type:uuid;default:gen_random_uuid()"`
	QuestionText     string    `gorm:"column:questionnaire_question_text;type:text;not null"`
	QuestionType     int       `gorm:"column:questionnaire_question_type;not null"` // 1=rating, 2=text, 3=choice
	QuestionOptions  []string  `gorm:"column:questionnaire_question_options;type:text[]"`
	EventID          *string   `gorm:"column:questionnaire_question_event_id;type:uuid"`
	LectureSessionID *string   `gorm:"column:questionnaire_question_lecture_session_id;type:uuid"`
	QuestionScope    int       `gorm:"column:questionnaire_question_scope;not null;default:1"` // 1=general, 2=event, 3=lecture
	CreatedAt        time.Time `gorm:"column:questionnaire_question_created_at;autoCreateTime"`

	// Optional relation
	Event          *EventModel.EventModel                   `gorm:"foreignKey:EventID"`
	LectureSession *LectureSessionModel.LectureSessionModel `gorm:"foreignKey:LectureSessionID"`
}

func (QuestionnaireQuestionModel) TableName() string {
	return "questionnaire_questions"
}
