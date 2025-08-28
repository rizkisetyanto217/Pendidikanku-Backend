package model

import (
	"time"

	"gorm.io/gorm"
)

type LectureSessionsQuizModel struct {
	LectureSessionsQuizID               string         `gorm:"column:lecture_sessions_quiz_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"lecture_sessions_quiz_id"`
	LectureSessionsQuizTitle            string         `gorm:"column:lecture_sessions_quiz_title;type:varchar(255);not null" json:"lecture_sessions_quiz_title"`
	LectureSessionsQuizDescription      *string        `gorm:"column:lecture_sessions_quiz_description;type:text" json:"lecture_sessions_quiz_description,omitempty"`
	LectureSessionsQuizLectureSessionID string         `gorm:"column:lecture_sessions_quiz_lecture_session_id;type:uuid;not null" json:"lecture_sessions_quiz_lecture_session_id"`
	LectureSessionsQuizMasjidID         string         `gorm:"column:lecture_sessions_quiz_masjid_id;type:uuid;not null" json:"lecture_sessions_quiz_masjid_id"`

	// timestamps
	LectureSessionsQuizCreatedAt time.Time      `gorm:"column:lecture_sessions_quiz_created_at;autoCreateTime" json:"lecture_sessions_quiz_created_at"`
	LectureSessionsQuizUpdatedAt time.Time      `gorm:"column:lecture_sessions_quiz_updated_at;autoUpdateTime" json:"lecture_sessions_quiz_updated_at"`
	LectureSessionsQuizDeletedAt gorm.DeletedAt `gorm:"column:lecture_sessions_quiz_deleted_at;index" json:"-"`

	// Kolom generated di DB (tsvector) – tidak perlu dimapping, cukup diabaikan
	// LectureSessionsQuizSearchTSV string `gorm:"column:lecture_sessions_quiz_search_tsv;type:tsvector" json:"-"`
}

// Nama tabel
func (LectureSessionsQuizModel) TableName() string {
	return "lecture_sessions_quiz"
}
