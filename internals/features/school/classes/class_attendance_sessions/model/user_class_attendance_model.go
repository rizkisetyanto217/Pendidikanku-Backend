// file: internals/features/attendance/model/user_class_session_attendance_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserClassSessionAttendanceModel struct {
	UserClassSessionAttendanceID               uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:user_class_session_attendance_id" json:"user_class_session_attendance_id"`
	UserClassSessionAttendanceMasjidID         uuid.UUID      `gorm:"type:uuid;not null;column:user_class_session_attendance_masjid_id"                    json:"user_class_session_attendance_masjid_id"`
	UserClassSessionAttendanceSessionID        uuid.UUID      `gorm:"type:uuid;not null;column:user_class_session_attendance_session_id"                   json:"user_class_session_attendance_session_id"`
	UserClassSessionAttendanceMasjidStudentID  uuid.UUID      `gorm:"type:uuid;not null;column:user_class_session_attendance_masjid_student_id"            json:"user_class_session_attendance_masjid_student_id"`

	UserClassSessionAttendanceStatus           string         `gorm:"type:varchar(16);not null;default:'unmarked';column:user_class_session_attendance_status" json:"user_class_session_attendance_status"`

	UserClassSessionAttendanceTypeID           *uuid.UUID     `gorm:"type:uuid;column:user_class_session_attendance_type_id"                                json:"user_class_session_attendance_type_id,omitempty"`
	UserClassSessionAttendanceDesc             *string        `gorm:"type:text;column:user_class_session_attendance_desc"                                   json:"user_class_session_attendance_desc,omitempty"`
	UserClassSessionAttendanceScore            *float64       `gorm:"type:numeric(5,2);column:user_class_session_attendance_score"                         json:"user_class_session_attendance_score,omitempty"`
	UserClassSessionAttendanceIsPassed         *bool          `gorm:"column:user_class_session_attendance_is_passed"                                       json:"user_class_session_attendance_is_passed,omitempty"`

	UserClassSessionAttendanceMarkedAt         *time.Time     `gorm:"type:timestamptz;column:user_class_session_attendance_marked_at"                      json:"user_class_session_attendance_marked_at,omitempty"`
	UserClassSessionAttendanceMarkedByTeacherID *uuid.UUID    `gorm:"type:uuid;column:user_class_session_attendance_marked_by_teacher_id"                  json:"user_class_session_attendance_marked_by_teacher_id,omitempty"`

	UserClassSessionAttendanceMethod           *string        `gorm:"type:varchar(16);column:user_class_session_attendance_method"                          json:"user_class_session_attendance_method,omitempty"`

	UserClassSessionAttendanceLat              *float64       `gorm:"column:user_class_session_attendance_lat"                                            json:"user_class_session_attendance_lat,omitempty"`
	UserClassSessionAttendanceLng              *float64       `gorm:"column:user_class_session_attendance_lng"                                            json:"user_class_session_attendance_lng,omitempty"`
	UserClassSessionAttendanceDistanceM        *int           `gorm:"column:user_class_session_attendance_distance_m"                                      json:"user_class_session_attendance_distance_m,omitempty"`

	UserClassSessionAttendanceLateSeconds      *int           `gorm:"column:user_class_session_attendance_late_seconds"                                    json:"user_class_session_attendance_late_seconds,omitempty"`

	UserClassSessionAttendanceUserNote         *string        `gorm:"type:text;column:user_class_session_attendance_user_note"                              json:"user_class_session_attendance_user_note,omitempty"`
	UserClassSessionAttendanceTeacherNote      *string        `gorm:"type:text;column:user_class_session_attendance_teacher_note"                           json:"user_class_session_attendance_teacher_note,omitempty"`

	UserClassSessionAttendanceLockedAt         *time.Time     `gorm:"type:timestamptz;column:user_class_session_attendance_locked_at"                       json:"user_class_session_attendance_locked_at,omitempty"`

	UserClassSessionAttendanceCreatedAt        time.Time      `gorm:"type:timestamptz;not null;default:now();column:user_class_session_attendance_created_at" json:"user_class_session_attendance_created_at"`
	UserClassSessionAttendanceUpdatedAt        time.Time      `gorm:"type:timestamptz;not null;default:now();column:user_class_session_attendance_updated_at" json:"user_class_session_attendance_updated_at"`
	UserClassSessionAttendanceDeletedAt        gorm.DeletedAt `gorm:"column:user_class_session_attendance_deleted_at;index"                                  json:"user_class_session_attendance_deleted_at,omitempty"`
}

func (UserClassSessionAttendanceModel) TableName() string {
	return "user_class_session_attendances"
}
