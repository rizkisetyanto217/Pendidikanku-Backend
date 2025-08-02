package model

import (
	"time"

	"github.com/google/uuid"
)

type UserLectureSessionsAttendanceModel struct {
	UserLectureSessionsAttendanceID             uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"user_lecture_sessions_attendance_id"`
	UserLectureSessionsAttendanceUserID         uuid.UUID  `gorm:"type:uuid;not null" json:"user_lecture_sessions_attendance_user_id"`
	UserLectureSessionsAttendanceLectureSessionID uuid.UUID `gorm:"type:uuid;not null" json:"user_lecture_sessions_attendance_lecture_session_id"`
	UserLectureSessionsAttendanceLectureID         uuid.UUID  `gorm:"type:uuid;not null" json:"user_lecture_sessions_attendance_lecture_id"`
	UserLectureSessionsAttendanceStatus int `gorm:"type:int" json:"user_lecture_sessions_attendance_status"`
	UserLectureSessionsAttendanceNotes          string     `gorm:"type:text" json:"user_lecture_sessions_attendance_notes"`
	UserLectureSessionsAttendancePersonalNotes  string     `gorm:"type:text" json:"user_lecture_sessions_attendance_personal_notes"`
	UserLectureSessionsAttendanceCreatedAt      time.Time  `gorm:"autoCreateTime" json:"user_lecture_sessions_attendance_created_at"`
	UserLectureSessionsAttendanceUpdatedAt      time.Time  `gorm:"autoUpdateTime" json:"user_lecture_sessions_attendance_updated_at"`
	UserLectureSessionsAttendanceDeletedAt      *time.Time `gorm:"index" json:"user_lecture_sessions_attendance_deleted_at,omitempty"`
}

func (UserLectureSessionsAttendanceModel) TableName() string {
	return "user_lecture_sessions_attendance"
}
