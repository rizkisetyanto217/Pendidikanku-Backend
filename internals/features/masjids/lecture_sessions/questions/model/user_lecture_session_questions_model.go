package model

import "time"

type LectureSessionsUserQuestionModel struct {
	LectureSessionsUserQuestionID        string    `gorm:"column:lecture_sessions_user_question_id;primaryKey;type:uuid;default:gen_random_uuid()"`
	LectureSessionsUserQuestionAnswer    string    `gorm:"column:lecture_sessions_user_question_answer;type:char(1);not null"` // A/B/C/D
	LectureSessionsUserQuestionIsCorrect bool      `gorm:"column:lecture_sessions_user_question_is_correct;not null"`
	LectureSessionsUserQuestionQuestionID string   `gorm:"column:lecture_sessions_user_question_question_id;type:uuid;not null"`
	LectureSessionsUserQuestionCreatedAt time.Time `gorm:"column:lecture_sessions_user_question_created_at;autoCreateTime"`

	// Optional: relation ke soal
	// Question *LectureSessionsQuestionModel `gorm:"foreignKey:LectureSessionsUserQuestionQuestionID"`
}

func (LectureSessionsUserQuestionModel) TableName() string {
	return "lecture_sessions_user_questions"
}
