package model

import (
	"time"
)

type LectureSessionsQuizModel struct {
	LectureSessionsQuizID               string    `gorm:"column:lecture_sessions_quiz_id;primaryKey;type:uuid;default:gen_random_uuid()" json:"lecture_sessions_quiz_id"`
	LectureSessionsQuizTitle            string    `gorm:"column:lecture_sessions_quiz_title;type:varchar(255);not null" json:"lecture_sessions_quiz_title"`
	LectureSessionsQuizDescription      string    `gorm:"column:lecture_sessions_quiz_description;type:text" json:"lecture_sessions_quiz_description"`
	LectureSessionsQuizLectureSessionID string    `gorm:"column:lecture_sessions_quiz_lecture_session_id;type:uuid;not null" json:"lecture_sessions_quiz_lecture_session_id"`
	LectureSessionsQuizCreatedAt        time.Time `gorm:"column:lecture_sessions_quiz_created_at;autoCreateTime" json:"lecture_sessions_quiz_created_at"`
}

func (LectureSessionsQuizModel) TableName() string {
	return "lecture_sessions_quiz"
}
