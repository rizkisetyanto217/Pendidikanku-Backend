package model

import (
	UserModel "masjidku_backend/internals/features/users/user/model"
	"time"
)

type UserLectureSessionModel struct {
	UserLectureSessionID               string     `gorm:"column:user_lecture_session_id;primaryKey;type:uuid;default:gen_random_uuid()"`
	UserLectureSessionAttendanceStatus int        `gorm:"column:user_lecture_session_attendance_status"` // 0 = tidak hadir, 1 = hadir, 2 = hadir online
	UserLectureSessionGradeResult      *float64   `gorm:"column:user_lecture_session_grade_result"`      // nullable

	// 📝 Catatan pembelajaran
	UserLectureSessionNotes            *string    `gorm:"column:user_lecture_session_notes" json:"user_lecture_session_notes"`

	UserLectureSessionLectureSessionID string     `gorm:"column:user_lecture_session_lecture_session_id;type:uuid;not null"`
	UserLectureSessionUserID           string     `gorm:"column:user_lecture_session_user_id;type:uuid;not null"`
	UserLectureSessionLectureID string `gorm:"column:user_lecture_session_lecture_id;type:uuid;not null" json:"user_lecture_session_lecture_id"`

	// 🕌 Masjid ID (baru)
	UserLectureSessionMasjidID         string     `gorm:"column:user_lecture_session_masjid_id;type:uuid;not null"`

	UserLectureSessionCreatedAt        time.Time  `gorm:"column:user_lecture_session_created_at;autoCreateTime"`

	UserLectureSessionUpdatedAt *time.Time `gorm:"column:user_lecture_session_updated_at" json:"user_lecture_session_updated_at"`

	// Relations
	User           *UserModel.UserModel   `gorm:"foreignKey:UserLectureSessionUserID"`
	LectureSession *LectureSessionModel   `gorm:"foreignKey:UserLectureSessionLectureSessionID"`
}

// TableName overrides the default table name
func (UserLectureSessionModel) TableName() string {
	return "user_lecture_sessions"
}
