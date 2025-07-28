package model

import (
	"time"
)

type UserLectureSessionsQuizModel struct {
	UserLectureSessionsQuizID        string    `gorm:"column:user_lecture_sessions_quiz_id;primaryKey;type:uuid;default:gen_random_uuid()" json:"user_lecture_sessions_quiz_id"`
	UserLectureSessionsQuizGrade     float64   `gorm:"column:user_lecture_sessions_quiz_grade_result" json:"user_lecture_sessions_quiz_grade_result"`
	UserLectureSessionsQuizQuizID    string    `gorm:"column:user_lecture_sessions_quiz_quiz_id;type:uuid;not null" json:"user_lecture_sessions_quiz_quiz_id"`
	UserLectureSessionsQuizUserID    string    `gorm:"column:user_lecture_sessions_quiz_user_id;type:uuid;not null" json:"user_lecture_sessions_quiz_user_id"`
	UserLectureSessionsQuizLectureSessionID string `gorm:"column:user_lecture_sessions_quiz_lecture_session_id;type:uuid;not null" json:"user_lecture_sessions_quiz_lecture_session_id"`
	// ✅ Count percobaan, default mulai dari 1
	UserLectureSessionsQuizAttemptCount int `gorm:"column:user_lecture_sessions_quiz_attempt_count;not null;default:1" json:"user_lecture_sessions_quiz_attempt_count"`

	UserLectureSessionsQuizDurationSeconds int `gorm:"column:user_lecture_sessions_quiz_duration_seconds" json:"user_lecture_sessions_quiz_duration_seconds"`

	// ✅ Tambahan masjid_id
	UserLectureSessionsQuizMasjidID  string    `gorm:"column:user_lecture_sessions_quiz_masjid_id;type:uuid;not null" json:"user_lecture_sessions_quiz_masjid_id"`

	UserLectureSessionsQuizCreatedAt time.Time `gorm:"column:user_lecture_sessions_quiz_created_at;autoCreateTime" json:"user_lecture_sessions_quiz_created_at"`
}

func (UserLectureSessionsQuizModel) TableName() string {
	return "user_lecture_sessions_quiz"
}
