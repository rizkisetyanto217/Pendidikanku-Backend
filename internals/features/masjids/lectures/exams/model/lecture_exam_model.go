package model

import (
	LectureModel "masjidku_backend/internals/features/masjids/lectures/main/model"
	"time"
)

type LectureExamModel struct {
	LectureExamID          string     `gorm:"column:lecture_exam_id;primaryKey;type:uuid;default:gen_random_uuid()" json:"lecture_exam_id"`
	LectureExamTitle       string     `gorm:"column:lecture_exam_title;type:varchar(255);not null" json:"lecture_exam_title"`
	LectureExamDescription *string    `gorm:"column:lecture_exam_description;type:text" json:"lecture_exam_description"`
	LectureExamLectureID   string     `gorm:"column:lecture_exam_lecture_id;type:uuid" json:"lecture_exam_lecture_id"`
	LectureExamMasjidID    string     `gorm:"column:lecture_exam_masjid_id;type:uuid;not null" json:"lecture_exam_masjid_id"`
	LectureExamCreatedAt   time.Time  `gorm:"column:lecture_exam_created_at;autoCreateTime" json:"lecture_exam_created_at"`

	// Relations
	Lecture *LectureModel.LectureModel `gorm:"foreignKey:LectureExamLectureID" json:"-"`
}

// TableName overrides the table name used by GORM
func (LectureExamModel) TableName() string {
	return "lecture_exams"
}
