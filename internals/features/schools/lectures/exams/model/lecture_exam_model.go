// ============================
// model/lecture_exam_model.go
// ============================

package model

import (
	"time"

	LectureModel "schoolku_backend/internals/features/schools/lectures/main/model"

	"gorm.io/gorm"
)

type LectureExamModel struct {
	LectureExamID          string  `gorm:"column:lecture_exam_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"lecture_exam_id"`
	LectureExamTitle       string  `gorm:"column:lecture_exam_title;type:varchar(255);not null" json:"lecture_exam_title"`
	LectureExamDescription *string `gorm:"column:lecture_exam_description;type:text" json:"lecture_exam_description,omitempty"`
	LectureExamLectureID   string  `gorm:"column:lecture_exam_lecture_id;type:uuid" json:"lecture_exam_lecture_id"`
	LectureExamSchoolID    string  `gorm:"column:lecture_exam_school_id;type:uuid;not null" json:"lecture_exam_school_id"`

	LectureExamCreatedAt time.Time      `gorm:"column:lecture_exam_created_at;autoCreateTime" json:"lecture_exam_created_at"`
	LectureExamUpdatedAt time.Time      `gorm:"column:lecture_exam_updated_at;autoUpdateTime" json:"lecture_exam_updated_at"`
	LectureExamDeletedAt gorm.DeletedAt `gorm:"column:lecture_exam_deleted_at;index" json:"-"`

	// Relations
	Lecture *LectureModel.LectureModel `gorm:"foreignKey:LectureExamLectureID" json:"-"`
}

func (LectureExamModel) TableName() string { return "lecture_exams" }
