package model

import (
	UserModel "masjidku_backend/internals/features/users/user/model"
	"time"
)

type UserLectureSessionModel struct {
	UserLectureSessionID               string     `gorm:"column:user_lecture_session_id;primaryKey;type:uuid;default:gen_random_uuid()"`
	UserLectureSessionAttendanceStatus int        `gorm:"column:user_lecture_session_attendance_status"` // 0 = tidak hadir, 1 = hadir, 2 = hadir online
	UserLectureSessionGradeResult      *float64   `gorm:"column:user_lecture_session_grade_result"`      // nullable
	UserLectureSessionLectureSessionID string     `gorm:"column:user_lecture_session_lecture_session_id;type:uuid;not null"`
	UserLectureSessionUserID           string     `gorm:"column:user_lecture_session_user_id;type:uuid;not null"`
	UserLectureSessionIsRegistered     bool       `gorm:"column:user_lecture_session_is_registered;default:false"`
	UserLectureSessionHasPaid          bool       `gorm:"column:user_lecture_session_has_paid;default:false"`
	UserLectureSessionPaidAmount       *int       `gorm:"column:user_lecture_session_paid_amount"`
	UserLectureSessionPaymentTime      *time.Time `gorm:"column:user_lecture_session_payment_time"`
	UserLectureSessionCreatedAt        time.Time  `gorm:"column:user_lecture_session_created_at;autoCreateTime"`

	// Relations
	User           *UserModel.UserModel `gorm:"foreignKey:UserLectureSessionUserID"`
	LectureSession *LectureSessionModel `gorm:"foreignKey:UserLectureSessionLectureSessionID"`
}

// TableName overrides the default table name
func (UserLectureSessionModel) TableName() string {
	return "user_lecture_sessions"
}
