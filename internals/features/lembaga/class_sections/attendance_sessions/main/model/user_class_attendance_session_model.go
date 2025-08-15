// internals/features/lembaga/classes/attendance/model/user_class_attendance_session_model.go
package model

import (
	"database/sql/driver"
	"strings"
	"time"

	"github.com/google/uuid"
)

/*
Status valid (TEXT):
- "present"
- "sick"
- "leave"
- "absent"
*/
type AttendanceStatus string

const (
	AttendancePresent AttendanceStatus = "present"
	AttendanceSick    AttendanceStatus = "sick"
	AttendanceLeave   AttendanceStatus = "leave"
	AttendanceAbsent  AttendanceStatus = "absent"
)

// Optional: jaga agar selalu lower-case saat scan/save
func (s *AttendanceStatus) Scan(value any) error {
	switch v := value.(type) {
	case string:
		*s = AttendanceStatus(strings.ToLower(strings.TrimSpace(v)))
	case []byte:
		*s = AttendanceStatus(strings.ToLower(strings.TrimSpace(string(v))))
	case nil:
		*s = ""
	default:
		*s = AttendanceStatus(strings.ToLower(strings.TrimSpace(v.(string))))
	}
	return nil
}

func (s AttendanceStatus) Value() (driver.Value, error) {
	return string(AttendanceStatus(strings.ToLower(strings.TrimSpace(string(s))))), nil
}

type UserClassAttendanceSessionModel struct {
	// PK
	UserClassAttendanceSessionsID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:user_class_attendance_sessions_id" json:"user_class_attendance_sessions_id"`

	// FK & Unique guard (per session_id + user_class_id)
	UserClassAttendanceSessionsSessionID    uuid.UUID `gorm:"type:uuid;not null;column:user_class_attendance_sessions_session_id;index:idx_ucas_session_status,priority:1;index:idx_ucas_masjid_session,priority:2;uniqueIndex:uq_ucas_session_userclass,priority:1" json:"user_class_attendance_sessions_session_id"`
	UserClassAttendanceSessionsUserClassID  uuid.UUID `gorm:"type:uuid;not null;column:user_class_attendance_sessions_user_class_id;index:idx_ucas_userclass_created_at,priority:1;uniqueIndex:uq_ucas_session_userclass,priority:2" json:"user_class_attendance_sessions_user_class_id"`
	UserClassAttendanceSessionsMasjidID     uuid.UUID `gorm:"type:uuid;not null;column:user_class_attendance_sessions_masjid_id;index:idx_ucas_masjid_created_at,priority:1;index:idx_ucas_masjid_session,priority:1" json:"user_class_attendance_sessions_masjid_id"`

	// TEXT status ('present'|'sick'|'leave'|'absent')
	UserClassAttendanceSessionsAttendanceStatus AttendanceStatus `gorm:"type:text;not null;column:user_class_attendance_sessions_attendance_status;index:idx_ucas_session_status,priority:2" json:"user_class_attendance_sessions_attendance_status"`

	// Skor 0..100 (nullable)
	UserClassAttendanceSessionsScore *int `gorm:"column:user_class_attendance_sessions_score" json:"user_class_attendance_sessions_score,omitempty"`

	// Lulus/tidak (nullable)
	UserClassAttendanceSessionsGradePassed *bool `gorm:"column:user_class_attendance_sessions_grade_passed" json:"user_class_attendance_sessions_grade_passed,omitempty"`

	// Catatan/materi (opsional)
	UserClassAttendanceSessionsMaterialPersonal *string `gorm:"column:user_class_attendance_sessions_material_personal" json:"user_class_attendance_sessions_material_personal,omitempty"`
	UserClassAttendanceSessionsPersonalNote     *string `gorm:"column:user_class_attendance_sessions_personal_note" json:"user_class_attendance_sessions_personal_note,omitempty"`
	UserClassAttendanceSessionsMemorization     *string `gorm:"column:user_class_attendance_sessions_memorization" json:"user_class_attendance_sessions_memorization,omitempty"`
	UserClassAttendanceSessionsHomework         *string `gorm:"column:user_class_attendance_sessions_homework" json:"user_class_attendance_sessions_homework,omitempty"`

	// Generated tsvector (read-only)
	UserClassAttendanceSessionsSearch *string `gorm:"type:tsvector;column:user_class_attendance_sessions_search;->" json:"-"`

	// Timestamps
	UserClassAttendanceSessionsCreatedAt time.Time  `gorm:"column:user_class_attendance_sessions_created_at;autoCreateTime" json:"user_class_attendance_sessions_created_at"`
	UserClassAttendanceSessionsUpdatedAt *time.Time `gorm:"column:user_class_attendance_sessions_updated_at;autoUpdateTime" json:"user_class_attendance_sessions_updated_at,omitempty"`
}

func (UserClassAttendanceSessionModel) TableName() string {
	return "user_class_attendance_sessions"
}
