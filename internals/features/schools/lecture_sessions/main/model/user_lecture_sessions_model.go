package model

import (
	"time"

	UserModel "schoolku_backend/internals/features/users/users/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserLectureSessionModel struct {
	UserLectureSessionID uuid.UUID `gorm:"column:user_lecture_session_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"user_lecture_session_id"`

	// Nilai/hasil (nullable, 0..100 divalidasi di DB)
	UserLectureSessionGradeResult *float64 `gorm:"column:user_lecture_session_grade_result" json:"user_lecture_session_grade_result"`

	// Relasi (UUID)
	UserLectureSessionLectureSessionID uuid.UUID `gorm:"column:user_lecture_session_lecture_session_id;type:uuid;not null" json:"user_lecture_session_lecture_session_id"`
	UserLectureSessionUserID           uuid.UUID `gorm:"column:user_lecture_session_user_id;type:uuid;not null" json:"user_lecture_session_user_id"`
	UserLectureSessionLectureID        uuid.UUID `gorm:"column:user_lecture_session_lecture_id;type:uuid;not null" json:"user_lecture_session_lecture_id"`

	// School cache
	UserLectureSessionSchoolID uuid.UUID `gorm:"column:user_lecture_session_school_id;type:uuid;not null" json:"user_lecture_session_school_id"`

	// Timestamps
	UserLectureSessionCreatedAt time.Time      `gorm:"column:user_lecture_session_created_at;autoCreateTime" json:"user_lecture_session_created_at"`
	UserLectureSessionUpdatedAt time.Time      `gorm:"column:user_lecture_session_updated_at;autoUpdateTime" json:"user_lecture_session_updated_at"`
	UserLectureSessionDeletedAt gorm.DeletedAt `gorm:"column:user_lecture_session_deleted_at;index" json:"user_lecture_session_deleted_at,omitempty"`

	// Relations (opsional dipakai saat Preload)
	User           *UserModel.UserModel `gorm:"foreignKey:UserLectureSessionUserID;references:id" json:"user,omitempty"`
	LectureSession *LectureSessionModel `gorm:"foreignKey:UserLectureSessionLectureSessionID;references:LectureSessionID" json:"lecture_session,omitempty"`
}

func (UserLectureSessionModel) TableName() string {
	return "user_lecture_sessions"
}
