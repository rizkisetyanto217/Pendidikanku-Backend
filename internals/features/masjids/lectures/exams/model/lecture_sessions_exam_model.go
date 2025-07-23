package model

import (
	LectureModel "masjidku_backend/internals/features/masjids/lectures/main/model"
	"time"
)

type LectureSessionsExamModel struct {
	LectureSessionsExamID          string     `gorm:"column:lecture_sessions_exam_id;primaryKey;type:uuid;default:gen_random_uuid()" json:"lecture_sessions_exam_id"`
	LectureSessionsExamTitle       string     `gorm:"column:lecture_sessions_exam_title;type:varchar(255);not null" json:"lecture_sessions_exam_title"`
	LectureSessionsExamDescription *string    `gorm:"column:lecture_sessions_exam_description;type:text" json:"lecture_sessions_exam_description"`
	LectureSessionsExamLectureID   string     `gorm:"column:lecture_sessions_exam_lecture_id;type:uuid" json:"lecture_sessions_exam_lecture_id"`
	LectureSessionsExamMasjidID    string     `gorm:"column:lecture_sessions_exam_masjid_id;type:uuid;not null" json:"lecture_sessions_exam_masjid_id"`
	LectureSessionsExamCreatedAt   time.Time  `gorm:"column:lecture_sessions_exam_created_at;autoCreateTime" json:"lecture_sessions_exam_created_at"`

	// Relations
	Lecture *LectureModel.LectureModel `gorm:"foreignKey:LectureSessionsExamLectureID" json:"-"`
}

// TableName overrides the table name used by GORM
func (LectureSessionsExamModel) TableName() string {
	return "lecture_sessions_exams"
}
