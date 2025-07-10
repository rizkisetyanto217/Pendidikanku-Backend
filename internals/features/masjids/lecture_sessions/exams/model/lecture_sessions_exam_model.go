package model

import (
	LectureModel "masjidku_backend/internals/features/masjids/lectures/model"
	"time"
)

type LectureSessionsExamModel struct {
	LectureSessionsExamID          string    `gorm:"column:lecture_sessions_exam_id;primaryKey;type:uuid;default:gen_random_uuid()"`
	LectureSessionsExamTitle       string    `gorm:"column:lecture_sessions_exam_title;type:varchar(255);not null"`
	LectureSessionsExamDescription *string   `gorm:"column:lecture_sessions_exam_description;type:text"`
	LectureSessionsExamLectureID   string    `gorm:"column:lecture_sessions_exam_lecture_id;type:uuid"`
	LectureSessionsExamCreatedAt   time.Time `gorm:"column:lecture_sessions_exam_created_at;autoCreateTime"`

	// Relations
	Lecture *LectureModel.LectureModel `gorm:"foreignKey:LectureSessionsExamLectureID"`
}

// TableName overrides the table name
func (LectureSessionsExamModel) TableName() string {
	return "lecture_sessions_exams"
}
