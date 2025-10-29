// file: internals/features/attendance/model/student_class_session_attendance_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type StudentClassSessionAttendanceModel struct {
	StudentClassSessionAttendanceID              uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:student_class_session_attendance_id" json:"student_class_session_attendance_id"`
	StudentClassSessionAttendanceMasjidID        uuid.UUID `gorm:"type:uuid;not null;column:student_class_session_attendance_masjid_id"                    json:"student_class_session_attendance_masjid_id"`
	StudentClassSessionAttendanceSessionID       uuid.UUID `gorm:"type:uuid;not null;column:student_class_session_attendance_session_id"                   json:"student_class_session_attendance_session_id"`
	StudentClassSessionAttendanceMasjidStudentID uuid.UUID `gorm:"type:uuid;not null;column:student_class_session_attendance_masjid_student_id"            json:"student_class_session_attendance_masjid_student_id"`

	StudentClassSessionAttendanceStatus string `gorm:"type:varchar(16);not null;default:'unmarked';column:student_class_session_attendance_status" json:"student_class_session_attendance_status"`

	StudentClassSessionAttendanceTypeID   *uuid.UUID `gorm:"type:uuid;column:student_class_session_attendance_type_id"                                json:"student_class_session_attendance_type_id,omitempty"`
	StudentClassSessionAttendanceDesc     *string    `gorm:"type:text;column:student_class_session_attendance_desc"                                   json:"student_class_session_attendance_desc,omitempty"`
	StudentClassSessionAttendanceScore    *float64   `gorm:"type:numeric(5,2);column:student_class_session_attendance_score"                         json:"student_class_session_attendance_score,omitempty"`
	StudentClassSessionAttendanceIsPassed *bool      `gorm:"column:student_class_session_attendance_is_passed"                                       json:"student_class_session_attendance_is_passed,omitempty"`

	StudentClassSessionAttendanceMarkedAt          *time.Time `gorm:"type:timestamptz;column:student_class_session_attendance_marked_at"                      json:"student_class_session_attendance_marked_at,omitempty"`
	StudentClassSessionAttendanceMarkedByTeacherID *uuid.UUID `gorm:"type:uuid;column:student_class_session_attendance_marked_by_teacher_id"                  json:"student_class_session_attendance_marked_by_teacher_id,omitempty"`

	StudentClassSessionAttendanceMethod *string `gorm:"type:varchar(16);column:student_class_session_attendance_method"                          json:"student_class_session_attendance_method,omitempty"`

	StudentClassSessionAttendanceLat       *float64 `gorm:"column:student_class_session_attendance_lat"                                            json:"student_class_session_attendance_lat,omitempty"`
	StudentClassSessionAttendanceLng       *float64 `gorm:"column:student_class_session_attendance_lng"                                            json:"student_class_session_attendance_lng,omitempty"`
	StudentClassSessionAttendanceDistanceM *int     `gorm:"column:student_class_session_attendance_distance_m"                                      json:"student_class_session_attendance_distance_m,omitempty"`

	StudentClassSessionAttendanceLateSeconds *int `gorm:"column:student_class_session_attendance_late_seconds"                                    json:"student_class_session_attendance_late_seconds,omitempty"`

	StudentClassSessionAttendanceUserNote    *string `gorm:"type:text;column:student_class_session_attendance_user_note"                              json:"student_class_session_attendance_user_note,omitempty"`
	StudentClassSessionAttendanceTeacherNote *string `gorm:"type:text;column:student_class_session_attendance_teacher_note"                           json:"student_class_session_attendance_teacher_note,omitempty"`

	StudentClassSessionAttendanceLockedAt *time.Time `gorm:"type:timestamptz;column:student_class_session_attendance_locked_at"                       json:"student_class_session_attendance_locked_at,omitempty"`

	StudentClassSessionAttendanceCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:student_class_session_attendance_created_at" json:"student_class_session_attendance_created_at"`
	StudentClassSessionAttendanceUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:student_class_session_attendance_updated_at" json:"student_class_session_attendance_updated_at"`
	StudentClassSessionAttendanceDeletedAt gorm.DeletedAt `gorm:"column:student_class_session_attendance_deleted_at;index"                                  json:"student_class_session_attendance_deleted_at,omitempty"`
}

func (StudentClassSessionAttendanceModel) TableName() string {
	return "student_class_session_attendances"
}
