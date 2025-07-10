package model

import (
	"time"

	"github.com/google/uuid"
)

type UserLectureModel struct {
	UserLectureID                     uuid.UUID `gorm:"column:user_lecture_id;primaryKey;type:uuid;default:gen_random_uuid()" json:"user_lecture_id"`
	UserLectureUserID                 uuid.UUID `gorm:"column:user_lecture_user_id;type:uuid;not null" json:"user_lecture_user_id"`
	UserLectureLectureID              uuid.UUID `gorm:"column:user_lecture_lecture_id;type:uuid;not null" json:"user_lecture_lecture_id"`
	UserLectureGrade                  int       `gorm:"column:user_lecture_grade_result" json:"user_lecture_grade_result"`
	UserLectureTotalCompletedSessions int       `gorm:"column:user_lecture_total_completed_sessions;default:0" json:"user_lecture_total_completed_sessions"`
	UserLectureCreatedAt time.Time `gorm:"column:user_lecture_created_at;autoCreateTime" json:"user_lecture_created_at"`
}

// TableName overrides the table name
func (UserLectureModel) TableName() string {
	return "user_lectures"
}
