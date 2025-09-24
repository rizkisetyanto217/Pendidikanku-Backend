// file: internals/features/school/attendance/sessions/model/class_attendance_session_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Selaraskan dengan enum session_status_enum di DB
type SessionStatus string

const (
	SessionStatusScheduled SessionStatus = "scheduled"
	SessionStatusOngoing   SessionStatus = "ongoing"
	SessionStatusCompleted SessionStatus = "completed"
	SessionStatusCanceled  SessionStatus = "canceled"
)

// Attendance status dari CHECK ('open','closed')
type AttendanceStatus string

const (
	AttendanceStatusOpen   AttendanceStatus = "open"
	AttendanceStatusClosed AttendanceStatus = "closed"
)

type ClassAttendanceSessionModel struct {
	// PK
	ClassAttendanceSessionID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:class_attendance_session_id" json:"class_attendance_session_id"`

	// Tenant & relasi utama
	ClassAttendanceSessionMasjidID   uuid.UUID  `gorm:"type:uuid;not null;column:class_attendance_session_masjid_id" json:"class_attendance_session_masjid_id"`
	ClassAttendanceSessionScheduleID uuid.UUID  `gorm:"type:uuid;not null;column:class_attendance_session_schedule_id" json:"class_attendance_session_schedule_id"`
	ClassAttendanceSessionRuleID     *uuid.UUID `gorm:"type:uuid;column:class_attendance_session_rule_id" json:"class_attendance_session_rule_id,omitempty"`

	// SLUG opsional
	ClassAttendanceSessionSlug *string `gorm:"type:varchar(160);column:class_attendance_session_slug" json:"class_attendance_session_slug,omitempty"`

	// Occurrence
	ClassAttendanceSessionDate     time.Time  `gorm:"type:date;not null;column:class_attendance_session_date" json:"class_attendance_session_date"`
	ClassAttendanceSessionStartsAt *time.Time `gorm:"type:timestamptz;column:class_attendance_session_starts_at" json:"class_attendance_session_starts_at,omitempty"`
	ClassAttendanceSessionEndsAt   *time.Time `gorm:"type:timestamptz;column:class_attendance_session_ends_at" json:"class_attendance_session_ends_at,omitempty"`

	// Lifecycle
	ClassAttendanceSessionStatus          SessionStatus    `gorm:"type:session_status_enum;not null;default:'scheduled';column:class_attendance_session_status" json:"class_attendance_session_status"`
	ClassAttendanceSessionAttendanceStatus AttendanceStatus `gorm:"type:text;not null;default:'open';column:class_attendance_session_attendance_status" json:"class_attendance_session_attendance_status"`
	ClassAttendanceSessionLocked          bool             `gorm:"not null;default:false;column:class_attendance_session_locked" json:"class_attendance_session_locked"`

	// Overrides
	ClassAttendanceSessionIsOverride        bool       `gorm:"not null;default:false;column:class_attendance_session_is_override" json:"class_attendance_session_is_override"`
	ClassAttendanceSessionIsCanceled        bool       `gorm:"not null;default:false;column:class_attendance_session_is_canceled" json:"class_attendance_session_is_canceled"`
	ClassAttendanceSessionOriginalStartAt   *time.Time `gorm:"type:timestamptz;column:class_attendance_session_original_start_at" json:"class_attendance_session_original_start_at,omitempty"`
	ClassAttendanceSessionOriginalEndAt     *time.Time `gorm:"type:timestamptz;column:class_attendance_session_original_end_at" json:"class_attendance_session_original_end_at,omitempty"`
	ClassAttendanceSessionKind              *string    `gorm:"type:text;column:class_attendance_session_kind" json:"class_attendance_session_kind,omitempty"`
	ClassAttendanceSessionOverrideReason    *string    `gorm:"type:text;column:class_attendance_session_override_reason" json:"class_attendance_session_override_reason,omitempty"`
	ClassAttendanceSessionOverrideEventID   *uuid.UUID `gorm:"type:uuid;column:class_attendance_session_override_event_id" json:"class_attendance_session_override_event_id,omitempty"`

	// Override resource (opsional)
	ClassAttendanceSessionTeacherID   *uuid.UUID `gorm:"type:uuid;column:class_attendance_session_teacher_id" json:"class_attendance_session_teacher_id,omitempty"`
	ClassAttendanceSessionClassRoomID *uuid.UUID `gorm:"type:uuid;column:class_attendance_session_class_room_id" json:"class_attendance_session_class_room_id,omitempty"`
	// Catatan: nama kolom di migration referensi ke table 'class_section_subject_teachers' – pastikan kolom rujukan konsisten.
	ClassAttendanceSessionCSSTID *uuid.UUID `gorm:"type:uuid;column:class_attendance_session_csst_id" json:"class_attendance_session_csst_id,omitempty"`

	// Info & rekap
	ClassAttendanceSessionTitle        *string `gorm:"type:text;column:class_attendance_session_title" json:"class_attendance_session_title,omitempty"`
	ClassAttendanceSessionGeneralInfo  string  `gorm:"type:text;not null;default:'';column:class_attendance_session_general_info" json:"class_attendance_session_general_info"`
	ClassAttendanceSessionNote         *string `gorm:"type:text;column:class_attendance_session_note" json:"class_attendance_session_note,omitempty"`
	ClassAttendanceSessionPresentCount *int    `gorm:"column:class_attendance_session_present_count" json:"class_attendance_session_present_count,omitempty"`
	ClassAttendanceSessionAbsentCount  *int    `gorm:"column:class_attendance_session_absent_count" json:"class_attendance_session_absent_count,omitempty"`
	ClassAttendanceSessionLateCount    *int    `gorm:"column:class_attendance_session_late_count" json:"class_attendance_session_late_count,omitempty"`
	ClassAttendanceSessionExcusedCount *int    `gorm:"column:class_attendance_session_excused_count" json:"class_attendance_session_excused_count,omitempty"`
	ClassAttendanceSessionSickCount    *int    `gorm:"column:class_attendance_session_sick_count" json:"class_attendance_session_sick_count,omitempty"`
	ClassAttendanceSessionLeaveCount   *int    `gorm:"column:class_attendance_session_leave_count" json:"class_attendance_session_leave_count,omitempty"`

	// Audit
	ClassAttendanceSessionCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:class_attendance_session_created_at" json:"class_attendance_session_created_at"`
	ClassAttendanceSessionUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:class_attendance_session_updated_at" json:"class_attendance_session_updated_at"`
	ClassAttendanceSessionDeletedAt gorm.DeletedAt `gorm:"column:class_attendance_session_deleted_at;index" json:"class_attendance_session_deleted_at,omitempty"`
}

func (ClassAttendanceSessionModel) TableName() string { return "class_attendance_sessions" }