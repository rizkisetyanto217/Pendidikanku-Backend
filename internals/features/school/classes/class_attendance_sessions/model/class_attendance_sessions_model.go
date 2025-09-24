// file: internals/features/school/class_attendance_sessions/model/class_attendance_session_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

/*
=========================================================

	Enums (mirror dari session_status_enum di DB)
	=========================================================
*/
type SessionStatus string

const (
	SessionScheduled SessionStatus = "scheduled"
	SessionOngoing   SessionStatus = "ongoing"
	SessionCompleted SessionStatus = "completed"
	SessionCanceled  SessionStatus = "canceled"
)

/*
=========================================================

	Model
	=========================================================
*/
type ClassAttendanceSessionModel struct {
	// PK
	ClassAttendanceSessionsID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:class_attendance_sessions_id" json:"class_attendance_sessions_id"`

	// Tenant guard
	ClassAttendanceSessionsMasjidID uuid.UUID `gorm:"type:uuid;not null;column:class_attendance_sessions_masjid_id" json:"class_attendance_sessions_masjid_id"`

	// Relasi utama: header jadwal (template)
	ClassAttendanceSessionsScheduleID uuid.UUID `gorm:"type:uuid;not null;column:class_attendance_sessions_schedule_id" json:"class_attendance_sessions_schedule_id"`

	// Jejak rule (opsional)
	ClassAttendanceSessionsRuleID *uuid.UUID `gorm:"type:uuid;column:class_attendance_sessions_rule_id" json:"class_attendance_sessions_rule_id,omitempty"`

	// SLUG (opsional; unik per tenant saat alive)
	ClassAttendanceSessionsSlug *string `gorm:"type:varchar(160);column:class_attendance_sessions_slug" json:"class_attendance_sessions_slug,omitempty"`

	// Occurrence
	ClassAttendanceSessionsDate     time.Time  `gorm:"type:date;not null;column:class_attendance_sessions_date" json:"class_attendance_sessions_date"`
	ClassAttendanceSessionsStartsAt *time.Time `gorm:"type:timestamptz;column:class_attendance_sessions_starts_at" json:"class_attendance_sessions_starts_at,omitempty"`
	ClassAttendanceSessionsEndsAt   *time.Time `gorm:"type:timestamptz;column:class_attendance_sessions_ends_at" json:"class_attendance_sessions_ends_at,omitempty"`

	// Lifecycle
	ClassAttendanceSessionsStatus           SessionStatus `gorm:"type:session_status_enum;not null;default:'scheduled';column:class_attendance_sessions_status" json:"class_attendance_sessions_status"`
	ClassAttendanceSessionsAttendanceStatus string        `gorm:"type:text;not null;default:'open';column:class_attendance_sessions_attendance_status" json:"class_attendance_sessions_attendance_status"`
	ClassAttendanceSessionsLocked           bool          `gorm:"not null;default:false;column:class_attendance_sessions_locked" json:"class_attendance_sessions_locked"`

	// Overrides (ubah harian)
	ClassAttendanceSessionsIsOverride      bool       `gorm:"not null;default:false;column:class_attendance_sessions_is_override" json:"class_attendance_sessions_is_override"`
	ClassAttendanceSessionsIsCanceled      bool       `gorm:"not null;default:false;column:class_attendance_sessions_is_canceled" json:"class_attendance_sessions_is_canceled"`
	ClassAttendanceSessionsOriginalStartAt *time.Time `gorm:"type:timestamptz;column:class_attendance_sessions_original_start_at" json:"class_attendance_sessions_original_start_at,omitempty"`
	ClassAttendanceSessionsOriginalEndAt   *time.Time `gorm:"type:timestamptz;column:class_attendance_sessions_original_end_at" json:"class_attendance_sessions_original_end_at,omitempty"`
	ClassAttendanceSessionsKind            *string    `gorm:"type:text;column:class_attendance_sessions_kind" json:"class_attendance_sessions_kind,omitempty"`
	ClassAttendanceSessionsOverrideReason  *string    `gorm:"type:text;column:class_attendance_sessions_override_reason" json:"class_attendance_sessions_override_reason,omitempty"`

	// Override karena EVENT (opsional)
	ClassAttendanceSessionsOverrideEventID           *uuid.UUID `gorm:"type:uuid;column:class_attendance_sessions_override_event_id" json:"class_attendance_sessions_override_event_id,omitempty"`
	ClassAttendanceSessionsOverrideAttendanceEventID *uuid.UUID `gorm:"type:uuid;column:class_attendance_sessions_override_attendance_event_id" json:"class_attendance_sessions_override_attendance_event_id,omitempty"`

	// Override resource (opsional)
	ClassAttendanceSessionsTeacherID   *uuid.UUID `gorm:"type:uuid;column:class_attendance_sessions_teacher_id" json:"class_attendance_sessions_teacher_id,omitempty"`
	ClassAttendanceSessionsClassRoomID *uuid.UUID `gorm:"type:uuid;column:class_attendance_sessions_class_room_id" json:"class_attendance_sessions_class_room_id,omitempty"`
	ClassAttendanceSessionsCSSTID      *uuid.UUID `gorm:"type:uuid;column:class_attendance_sessions_csst_id" json:"class_attendance_sessions_csst_id,omitempty"`

	// Info & rekap
	ClassAttendanceSessionsTitle       *string `gorm:"type:text;column:class_attendance_sessions_title" json:"class_attendance_sessions_title,omitempty"`
	ClassAttendanceSessionsGeneralInfo string  `gorm:"type:text;not null;default:'';column:class_attendance_sessions_general_info" json:"class_attendance_sessions_general_info"`
	ClassAttendanceSessionsNote        *string `gorm:"type:text;column:class_attendance_sessions_note" json:"class_attendance_sessions_note,omitempty"`

	ClassAttendanceSessionsPresentCount *int `gorm:"column:class_attendance_sessions_present_count" json:"class_attendance_sessions_present_count,omitempty"`
	ClassAttendanceSessionsAbsentCount  *int `gorm:"column:class_attendance_sessions_absent_count" json:"class_attendance_sessions_absent_count,omitempty"`
	ClassAttendanceSessionsLateCount    *int `gorm:"column:class_attendance_sessions_late_count" json:"class_attendance_sessions_late_count,omitempty"`
	ClassAttendanceSessionsExcusedCount *int `gorm:"column:class_attendance_sessions_excused_count" json:"class_attendance_sessions_excused_count,omitempty"`
	ClassAttendanceSessionsSickCount    *int `gorm:"column:class_attendance_sessions_sick_count" json:"class_attendance_sessions_sick_count,omitempty"`
	ClassAttendanceSessionsLeaveCount   *int `gorm:"column:class_attendance_sessions_leave_count" json:"class_attendance_sessions_leave_count,omitempty"`

	// Audit & soft delete
	ClassAttendanceSessionsCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();autoCreateTime;column:class_attendance_sessions_created_at" json:"class_attendance_sessions_created_at"`
	ClassAttendanceSessionsUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();autoUpdateTime;column:class_attendance_sessions_updated_at" json:"class_attendance_sessions_updated_at"`
	ClassAttendanceSessionsDeletedAt gorm.DeletedAt `gorm:"column:class_attendance_sessions_deleted_at;index" json:"class_attendance_sessions_deleted_at,omitempty"`
}

func (ClassAttendanceSessionModel) TableName() string { return "class_attendance_sessions" }
