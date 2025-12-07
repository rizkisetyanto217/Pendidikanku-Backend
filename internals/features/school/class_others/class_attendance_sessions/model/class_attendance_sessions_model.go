// file: internals/features/school/classes/class_attendance_sessions/model/class_attendance_session_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

/* =========================
   ENUMS (selaras dgn DB)
========================= */

type SessionStatus string

const (
	SessionStatusScheduled SessionStatus = "scheduled"
	SessionStatusOngoing   SessionStatus = "ongoing"
	SessionStatusCompleted SessionStatus = "completed"
	SessionStatusCanceled  SessionStatus = "canceled"
)

type AttendanceStatus string

const (
	AttendanceStatusOpen   AttendanceStatus = "open"
	AttendanceStatusClosed AttendanceStatus = "closed"
)

/* =========================================
   MODEL: class_attendance_sessions (simplified)
========================================= */

type ClassAttendanceSessionModel struct {
	// PK
	ClassAttendanceSessionID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:class_attendance_session_id" json:"class_attendance_session_id"`

	// Tenant & relasi utama
	ClassAttendanceSessionSchoolID   uuid.UUID  `gorm:"type:uuid;not null;column:class_attendance_session_school_id" json:"class_attendance_session_school_id"`
	ClassAttendanceSessionScheduleID *uuid.UUID `gorm:"type:uuid;column:class_attendance_session_schedule_id" json:"class_attendance_session_schedule_id,omitempty"`
	ClassAttendanceSessionRuleID     *uuid.UUID `gorm:"type:uuid;column:class_attendance_session_rule_id" json:"class_attendance_session_rule_id,omitempty"`

	// SLUG opsional
	ClassAttendanceSessionSlug *string `gorm:"type:varchar(160);column:class_attendance_session_slug" json:"class_attendance_session_slug,omitempty"`

	// Occurrence (tanggal + waktu)
	ClassAttendanceSessionDate     time.Time  `gorm:"type:date;not null;column:class_attendance_session_date" json:"class_attendance_session_date"`
	ClassAttendanceSessionStartsAt *time.Time `gorm:"type:timestamptz;column:class_attendance_session_starts_at" json:"class_attendance_session_starts_at,omitempty"`
	ClassAttendanceSessionEndsAt   *time.Time `gorm:"type:timestamptz;column:class_attendance_session_ends_at" json:"class_attendance_session_ends_at,omitempty"`

	// Lifecycle
	ClassAttendanceSessionStatus           SessionStatus    `gorm:"type:session_status_enum;not null;default:'scheduled';column:class_attendance_session_status" json:"class_attendance_session_status"`
	ClassAttendanceSessionAttendanceStatus AttendanceStatus `gorm:"type:text;not null;default:'open';column:class_attendance_session_attendance_status" json:"class_attendance_session_attendance_status"`
	ClassAttendanceSessionLocked           bool             `gorm:"not null;default:false;column:class_attendance_session_locked" json:"class_attendance_session_locked"`

	// Overrides
	ClassAttendanceSessionIsOverride      bool       `gorm:"not null;default:false;column:class_attendance_session_is_override" json:"class_attendance_session_is_override"`
	ClassAttendanceSessionIsCanceled      bool       `gorm:"not null;default:false;column:class_attendance_session_is_canceled" json:"class_attendance_session_is_canceled"`
	ClassAttendanceSessionOriginalStartAt *time.Time `gorm:"type:timestamptz;column:class_attendance_session_original_start_at" json:"class_attendance_session_original_start_at,omitempty"`
	ClassAttendanceSessionOriginalEndAt   *time.Time `gorm:"type:timestamptz;column:class_attendance_session_original_end_at" json:"class_attendance_session_original_end_at,omitempty"`
	ClassAttendanceSessionKind            *string    `gorm:"type:text;column:class_attendance_session_kind" json:"class_attendance_session_kind,omitempty"`
	ClassAttendanceSessionOverrideReason  *string    `gorm:"type:text;column:class_attendance_session_override_reason" json:"class_attendance_session_override_reason,omitempty"`
	ClassAttendanceSessionOverrideEventID *uuid.UUID `gorm:"type:uuid;column:class_attendance_session_override_event_id" json:"class_attendance_session_override_event_id,omitempty"`

	// Override resource (opsional)
	ClassAttendanceSessionTeacherID   *uuid.UUID `gorm:"type:uuid;column:class_attendance_session_teacher_id" json:"class_attendance_session_teacher_id,omitempty"`
	ClassAttendanceSessionClassRoomID *uuid.UUID `gorm:"type:uuid;column:class_attendance_session_class_room_id" json:"class_attendance_session_class_room_id,omitempty"`
	ClassAttendanceSessionCSSTID      *uuid.UUID `gorm:"type:uuid;column:class_attendance_session_csst_id" json:"class_attendance_session_csst_id,omitempty"`

	// Tipe sesi (master per tenant) + snapshot
	ClassAttendanceSessionTypeID       *uuid.UUID        `gorm:"type:uuid;column:class_attendance_session_type_id" json:"class_attendance_session_type_id,omitempty"`
	ClassAttendanceSessionTypeSnapshot datatypes.JSONMap `gorm:"type:jsonb;column:class_attendance_session_type_snapshot" json:"class_attendance_session_type_snapshot,omitempty"`

	// Info & rekap
	ClassAttendanceSessionTitle       *string `gorm:"type:text;column:class_attendance_session_title" json:"class_attendance_session_title,omitempty"`
	ClassAttendanceSessionGeneralInfo string  `gorm:"type:text;not null;default:'';column:class_attendance_session_general_info" json:"class_attendance_session_general_info"`
	ClassAttendanceSessionNote        *string `gorm:"type:text;column:class_attendance_session_note" json:"class_attendance_session_note,omitempty"`

	// Nomor pertemuan (meeting number)
	ClassAttendanceSessionMeetingNumber *int `gorm:"column:class_attendance_session_meeting_number" json:"class_attendance_session_meeting_number,omitempty"`

	// Rekap kehadiran
	ClassAttendanceSessionPresentCount *int `gorm:"column:class_attendance_session_present_count" json:"class_attendance_session_present_count,omitempty"`
	ClassAttendanceSessionAbsentCount  *int `gorm:"column:class_attendance_session_absent_count" json:"class_attendance_session_absent_count,omitempty"`
	ClassAttendanceSessionLateCount    *int `gorm:"column:class_attendance_session_late_count" json:"class_attendance_session_late_count,omitempty"`
	ClassAttendanceSessionExcusedCount *int `gorm:"column:class_attendance_session_excused_count" json:"class_attendance_session_excused_count,omitempty"`
	ClassAttendanceSessionSickCount    *int `gorm:"column:class_attendance_session_sick_count" json:"class_attendance_session_sick_count,omitempty"`
	ClassAttendanceSessionLeaveCount   *int `gorm:"column:class_attendance_session_leave_count" json:"class_attendance_session_leave_count,omitempty"`

	// Audit
	ClassAttendanceSessionCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:class_attendance_session_created_at" json:"class_attendance_session_created_at"`
	ClassAttendanceSessionUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:class_attendance_session_updated_at" json:"class_attendance_session_updated_at"`
	ClassAttendanceSessionDeletedAt gorm.DeletedAt `gorm:"column:class_attendance_session_deleted_at;index" json:"class_attendance_session_deleted_at,omitempty"`
}

func (ClassAttendanceSessionModel) TableName() string {
	return "class_attendance_sessions"
}
