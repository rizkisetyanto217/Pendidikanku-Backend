// internals/features/school/attendance_assesment/user_result/user_attendance/model/user_attendance_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserAttendanceStatus string

const (
	UserAttendancePresent UserAttendanceStatus = "present"
	UserAttendanceAbsent  UserAttendanceStatus = "absent"
	UserAttendanceExcused UserAttendanceStatus = "excused"
	UserAttendanceLate    UserAttendanceStatus = "late"
)

type UserAttendanceModel struct {
	// PK
	UserAttendanceID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:user_attendance_id" json:"user_attendance_id"`

	// FKs
	UserAttendanceMasjidID      uuid.UUID `gorm:"type:uuid;not null;column:user_attendance_masjid_id" json:"user_attendance_masjid_id"`
	UserAttendanceSessionID     uuid.UUID `gorm:"type:uuid;not null;column:user_attendance_session_id;index:idx_user_attendance_session" json:"user_attendance_session_id"`
	UserAttendanceMasjidStudentID uuid.UUID `gorm:"type:uuid;not null;column:user_attendance_masjid_student_id;index:idx_user_attendance_student" json:"user_attendance_masjid_student_id"`

	// Status (DB constraint via CHECK)
	UserAttendanceStatus UserAttendanceStatus `gorm:"type:varchar(16);not null;default:present;column:user_attendance_status;index:idx_user_attendance_status" json:"user_attendance_status"`

	// FK ke master type (nullable). DB: REFERENCES user_type(user_type_id) ON DELETE SET NULL
	UserAttendanceTypeID *uuid.UUID `gorm:"type:uuid;column:user_attendance_type_id;index:idx_user_attendance_type_id" json:"user_attendance_type_id,omitempty"`

	// Ringkasan catatan Qur'an harian
	UserAttendanceDesc     *string  `gorm:"type:text;column:user_attendance_desc" json:"user_attendance_desc,omitempty"`
	UserAttendanceScore    *float64 `gorm:"type:numeric(5,2);column:user_attendance_score" json:"user_attendance_score,omitempty"` // DB: CHECK 0..100
	UserAttendanceIsPassed *bool    `gorm:"column:user_attendance_is_passed" json:"user_attendance_is_passed,omitempty"`

	// Notes (nullable)
	UserAttendanceUserNote    *string `gorm:"type:text;column:user_attendance_user_note" json:"user_attendance_user_note,omitempty"`
	UserAttendanceTeacherNote *string `gorm:"type:text;column:user_attendance_teacher_note" json:"user_attendance_teacher_note,omitempty"`

	// Timestamps
	UserAttendanceCreatedAt time.Time      `gorm:"column:user_attendance_created_at;autoCreateTime" json:"user_attendance_created_at"`
	UserAttendanceUpdatedAt time.Time      `gorm:"column:user_attendance_updated_at;autoUpdateTime" json:"user_attendance_updated_at"`
	UserAttendanceDeletedAt gorm.DeletedAt `gorm:"column:user_attendance_deleted_at;index" json:"user_attendance_deleted_at,omitempty"`
}

func (UserAttendanceModel) TableName() string {
	return "user_attendance"
}
