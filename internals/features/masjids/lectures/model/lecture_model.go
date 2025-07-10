package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type LectureModel struct {
	LectureID                 uuid.UUID      `gorm:"column:lecture_id;primaryKey;type:uuid;default:gen_random_uuid()" json:"lecture_id"`
	LectureTitle              string         `gorm:"column:lecture_title;type:varchar(255);not null" json:"lecture_title"`
	LectureDescription        string         `gorm:"column:lecture_description;type:text" json:"lecture_description"`
	TotalLectureSessions      *int           `gorm:"column:total_lecture_sessions" json:"total_lecture_sessions,omitempty"`
	LectureIsRecurring        bool           `gorm:"column:lecture_is_recurring;default:false" json:"lecture_is_recurring"`
	LectureRecurrenceInterval *int           `gorm:"column:lecture_recurrence_interval" json:"lecture_recurrence_interval,omitempty"`
	LectureImageURL           *string        `gorm:"column:lecture_image_url;type:text" json:"lecture_image_url,omitempty"`
	LectureTeachers           datatypes.JSON `gorm:"column:lecture_teachers;type:jsonb" json:"lecture_teachers,omitempty"`
	LectureMasjidID           uuid.UUID      `gorm:"column:lecture_masjid_id;type:uuid;not null" json:"lecture_masjid_id"`
	LectureCreatedAt          time.Time      `gorm:"column:lecture_created_at;autoCreateTime" json:"lecture_created_at"`
}

func (LectureModel) TableName() string {
	return "lectures"
}
