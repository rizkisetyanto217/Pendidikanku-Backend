package model

import (
	"time"
)

type LectureSessionsQuestionModel struct {
	LectureSessionsQuestionID          string `gorm:"column:lecture_sessions_question_id;primaryKey;type:uuid;default:gen_random_uuid()" json:"lecture_sessions_question_id"`
	LectureSessionsQuestion            string `gorm:"column:lecture_sessions_question;type:text;not null" json:"lecture_sessions_question"`
	LectureSessionsQuestionAnswer      string `gorm:"column:lecture_sessions_question_answer;type:text;not null" json:"lecture_sessions_question_answer"`
	LectureSessionsQuestionCorrect     string `gorm:"column:lecture_sessions_question_correct;type:char(1);not null" json:"lecture_sessions_question_correct"` // A/B/C/D
	LectureSessionsQuestionExplanation string `gorm:"column:lecture_sessions_question_explanation;type:text" json:"lecture_sessions_question_explanation"`

	// Relasi opsional ke quiz atau exam
	LectureSessionsQuestionQuizID *string `gorm:"column:lecture_sessions_question_quiz_id;type:uuid" json:"lecture_sessions_question_quiz_id"`
	LectureSessionsQuestionExamID *string `gorm:"column:lecture_sessions_question_exam_id;type:uuid" json:"lecture_sessions_question_exam_id"`

	LectureSessionsQuestionCreatedAt time.Time `gorm:"column:lecture_sessions_question_created_at;autoCreateTime" json:"lecture_sessions_question_created_at"`
}

func (LectureSessionsQuestionModel) TableName() string {
	return "lecture_sessions_questions"
}
